package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
)

const (
	emailUniqueConstraint = "users_email_key"
	selectWithAlias       = "users.*, hero_profiles.alias AS alias"
	joinHeroProfile       = "LEFT JOIN hero_profiles ON hero_profiles.user_id = users.id"
	likeEscape            = `\`

	// maxFailedLoginCount est la valeur maximale du compteur (colonne smallint) ; maxLockoutExponent
	// plafonne le doublement du délai de verrouillage pour éviter tout dépassement de calcul.
	maxFailedLoginCount = 32767
	maxLockoutExponent  = 20
)

// LoginFailure est l'état d'un compte après un échec de connexion.
type LoginFailure struct {
	Count       int
	LockedUntil *time.Time
}

// Filter restreint la liste des comptes (modération). Les champs vides sont ignorés.
type Filter struct {
	Role        Role
	Status      Status
	EmailPrefix string
	Limit       int
	Offset      int
}

// Repository est le stockage des comptes. Les méthodes qui modifient un état sous condition
// retournent false quand la condition n'est pas remplie : la vérification et l'écriture se
// font dans une seule instruction SQL, sans fenêtre de concurrence.
type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindHeroByAlias(ctx context.Context, alias string) (*User, error)

	// Activate passe un compte de « en attente d'activation » à « actif ».
	Activate(ctx context.Context, id uuid.UUID, now time.Time) (bool, error)
	// UpdatePasswordHash remplace le mot de passe et lève tout verrouillage.
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error
	// RecordLoginFailure incrémente le compteur d'échecs de façon atomique et verrouille le
	// compte (délai croissant) une fois le seuil atteint.
	RecordLoginFailure(ctx context.Context, id uuid.UUID, now time.Time, lockout config.Lockout) (LoginFailure, error)
	RecordLoginSuccess(ctx context.Context, id uuid.UUID, now time.Time) error

	Deactivate(ctx context.Context, id uuid.UUID, now time.Time) (bool, error)
	Reactivate(ctx context.Context, id uuid.UUID, now time.Time) (bool, error)
	MarkVerified(ctx context.Context, id, verifiedBy uuid.UUID, now time.Time) (bool, error)
	SetRole(ctx context.Context, id uuid.UUID, role Role, now time.Time) error

	// UpdateContact remplace les coordonnées indiquées (téléphone, adresse) ; ErrNotFound si le
	// compte n'existe pas.
	UpdateContact(ctx context.Context, id uuid.UUID, update ContactUpdate, now time.Time) error
	// ChangeEmail remplace l'adresse de connexion. Il retourne ErrEmailTaken si une autre adresse
	// identique existe : l'unicité est garantie par la base, pas par une vérification préalable.
	ChangeEmail(ctx context.Context, id uuid.UUID, email string, now time.Time) error
	// SetAvatarPath remplace la photo de profil et retourne le chemin de la précédente (vide s'il n'y
	// en avait pas), pour que l'appelant puisse la supprimer.
	SetAvatarPath(ctx context.Context, id uuid.UUID, path string, now time.Time) (previous string, err error)

	List(ctx context.Context, filter Filter) (users []User, total int64, err error)
	// ActiveEmailsByRole retourne les adresses des comptes actifs ayant l'un des rôles.
	ActiveEmailsByRole(ctx context.Context, roles ...Role) ([]string, error)
}

type gormRepository struct {
	db     *gorm.DB
	sealer Sealer
}

// NewRepository crée le dépôt des comptes adossé à PostgreSQL.
func NewRepository(db *gorm.DB, sealer Sealer) Repository {
	return &gormRepository{db: db, sealer: sealer}
}

func (r *gormRepository) conn(ctx context.Context) *gorm.DB {
	return database.Conn(ctx, r.db)
}

func (r *gormRepository) Create(ctx context.Context, u *User) error {
	rec, err := toRecord(u, r.sealer)
	if err != nil {
		return err
	}
	if err := r.conn(ctx).Create(&rec).Error; err != nil {
		if database.IsUniqueViolation(err, emailUniqueConstraint) {
			return ErrEmailTaken
		}
		return fmt.Errorf("user : création : %w", err)
	}
	u.CreatedAt, u.UpdatedAt = rec.CreatedAt, rec.UpdatedAt
	return nil
}

func (r *gormRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return r.find(ctx, "users.id = ?", id)
}

func (r *gormRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	return r.find(ctx, "users.email = ?", email)
}

func (r *gormRepository) FindHeroByAlias(ctx context.Context, alias string) (*User, error) {
	return r.find(ctx, "hero_profiles.alias = ? AND users.role = ?", alias, string(RoleHero))
}

func (r *gormRepository) find(ctx context.Context, condition string, args ...any) (*User, error) {
	var found row
	err := r.conn(ctx).Table(tableUsers).
		Select(selectWithAlias).
		Joins(joinHeroProfile).
		Where(condition, args...).
		Take(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user : recherche : %w", err)
	}
	return found.toUser(r.sealer)
}

func (r *gormRepository) Activate(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	return r.conditionalUpdate(ctx, "id = ? AND status = ?", []any{id, string(StatusPendingActivation)}, map[string]any{
		"status":       string(StatusActive),
		"activated_at": now,
		"updated_at":   now,
	})
}

func (r *gormRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	changed, err := r.conditionalUpdate(ctx, "id = ?", []any{id}, map[string]any{
		"password_hash":      hash,
		"failed_login_count": 0,
		"locked_until":       nil,
		"updated_at":         now,
	})
	if err != nil {
		return err
	}
	if !changed {
		return ErrNotFound
	}
	return nil
}

// recordLoginFailureSQL incrémente le compteur et calcule le verrouillage dans une seule
// instruction : deux échecs simultanés ne peuvent pas s'écraser. Le délai double à chaque
// échec au-delà du seuil (exposant plafonné pour éviter tout dépassement), borné par maxSecs.
const recordLoginFailureSQL = `
UPDATE users
SET failed_login_count = LEAST(failed_login_count + 1, ?::int),
    locked_until = CASE
        WHEN failed_login_count + 1 >= ?::int
        THEN ?::timestamptz + make_interval(secs => LEAST(
            ?::float8 * power(2, LEAST(failed_login_count + 1 - ?::int, ?::int)),
            ?::float8))
        ELSE locked_until
    END,
    updated_at = ?::timestamptz
WHERE id = ?
RETURNING failed_login_count, locked_until`

func (r *gormRepository) RecordLoginFailure(ctx context.Context, id uuid.UUID, now time.Time, lockout config.Lockout) (LoginFailure, error) {
	var result struct {
		FailedLoginCount int
		LockedUntil      *time.Time
	}
	query := r.conn(ctx).Raw(recordLoginFailureSQL,
		maxFailedLoginCount,
		lockout.MaxFailures, now, lockout.BaseDelay.Seconds(), lockout.MaxFailures, maxLockoutExponent,
		lockout.MaxDelay.Seconds(), now, id,
	).Scan(&result)
	if query.Error != nil {
		return LoginFailure{}, fmt.Errorf("user : enregistrement de l'échec de connexion : %w", query.Error)
	}
	if query.RowsAffected == 0 {
		return LoginFailure{}, ErrNotFound
	}
	return LoginFailure{Count: result.FailedLoginCount, LockedUntil: result.LockedUntil}, nil
}

func (r *gormRepository) RecordLoginSuccess(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := r.conditionalUpdate(ctx, "id = ?", []any{id}, map[string]any{
		"failed_login_count": 0,
		"locked_until":       nil,
		"last_login_at":      now,
		"updated_at":         now,
	})
	return err
}

func (r *gormRepository) Deactivate(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	return r.conditionalUpdate(ctx, "id = ? AND status <> ?", []any{id, string(StatusDeactivated)}, map[string]any{
		"status":         string(StatusDeactivated),
		"deactivated_at": now,
		"updated_at":     now,
	})
}

// Reactivate rend le compte à son état d'avant la désactivation : actif s'il avait été
// activé, sinon en attente d'activation.
func (r *gormRepository) Reactivate(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	return r.conditionalUpdate(ctx, "id = ? AND status = ?", []any{id, string(StatusDeactivated)}, map[string]any{
		"status": gorm.Expr("CASE WHEN activated_at IS NULL THEN ? ELSE ? END",
			string(StatusPendingActivation), string(StatusActive)),
		"deactivated_at": nil,
		"updated_at":     now,
	})
}

func (r *gormRepository) MarkVerified(ctx context.Context, id, verifiedBy uuid.UUID, now time.Time) (bool, error) {
	return r.conditionalUpdate(ctx, "id = ? AND verified_at IS NULL", []any{id}, map[string]any{
		"verified_at": now,
		"verified_by": verifiedBy,
		"updated_at":  now,
	})
}

func (r *gormRepository) SetRole(ctx context.Context, id uuid.UUID, role Role, now time.Time) error {
	changed, err := r.conditionalUpdate(ctx, "id = ?", []any{id}, map[string]any{
		"role":       string(role),
		"updated_at": now,
	})
	if err != nil {
		return err
	}
	if !changed {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepository) UpdateContact(ctx context.Context, id uuid.UUID, update ContactUpdate, now time.Time) error {
	c := codec{sealer: r.sealer, id: id}
	values := map[string]any{"updated_at": now}

	set := func(column, value string) error {
		sealed, err := c.sealOptional(column, value)
		if err != nil {
			return err
		}
		if sealed == nil {
			values[column] = nil // une valeur vide s'enregistre NULL, comme à la création
		} else {
			values[column] = sealed
		}
		return nil
	}
	if update.Phone != nil {
		if err := set(columnPhone, *update.Phone); err != nil {
			return err
		}
	}
	if address := update.Address; address != nil {
		for column, value := range map[string]string{
			columnStreet: address.Street, columnPostalCode: address.PostalCode,
			columnCity: address.City, columnCountry: address.Country,
		} {
			if err := set(column, value); err != nil {
				return err
			}
		}
		if address.Complement != nil {
			if err := set(columnAddressComplement, *address.Complement); err != nil {
				return err
			}
		}
	}

	changed, err := r.conditionalUpdate(ctx, "id = ?", []any{id}, values)
	if err != nil {
		return err
	}
	if !changed {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepository) ChangeEmail(ctx context.Context, id uuid.UUID, email string, now time.Time) error {
	result := r.conn(ctx).Table(tableUsers).Where("id = ?", id).
		Updates(map[string]any{"email": email, "updated_at": now})
	if result.Error != nil {
		if database.IsUniqueViolation(result.Error, emailUniqueConstraint) {
			return ErrEmailTaken
		}
		return fmt.Errorf("user : changement d'adresse email : %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// setAvatarPathSQL remplace la photo et retourne l'ancienne dans une seule instruction : la ligne est
// verrouillée entre la lecture et l'écriture, donc deux remplacements simultanés ne peuvent pas
// retourner le même « ancien » chemin (et laisser un fichier orphelin).
const setAvatarPathSQL = `
WITH previous AS (SELECT avatar_path FROM users WHERE id = ? FOR UPDATE)
UPDATE users SET avatar_path = ?, updated_at = ?
FROM previous
WHERE users.id = ?
RETURNING previous.avatar_path`

func (r *gormRepository) SetAvatarPath(ctx context.Context, id uuid.UUID, path string, now time.Time) (string, error) {
	var result struct{ AvatarPath *string }
	query := r.conn(ctx).Raw(setAvatarPathSQL, id, path, now, id).Scan(&result)
	if query.Error != nil {
		return "", fmt.Errorf("user : remplacement de la photo : %w", query.Error)
	}
	if query.RowsAffected == 0 {
		return "", ErrNotFound
	}
	if result.AvatarPath == nil {
		return "", nil
	}
	return *result.AvatarPath, nil
}

func (r *gormRepository) conditionalUpdate(ctx context.Context, condition string, args []any, values map[string]any) (bool, error) {
	result := r.conn(ctx).Table(tableUsers).Where(condition, args...).Updates(values)
	if result.Error != nil {
		return false, fmt.Errorf("user : mise à jour : %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *gormRepository) List(ctx context.Context, filter Filter) ([]User, int64, error) {
	query := func() *gorm.DB {
		q := r.conn(ctx).Table(tableUsers).Joins(joinHeroProfile)
		if filter.Role != "" {
			q = q.Where("users.role = ?", string(filter.Role))
		}
		if filter.Status != "" {
			q = q.Where("users.status = ?", string(filter.Status))
		}
		if filter.EmailPrefix != "" {
			q = q.Where("users.email LIKE ? ESCAPE '"+likeEscape+"'", escapeLike(filter.EmailPrefix)+"%")
		}
		return q
	}

	var total int64
	if err := query().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("user : comptage : %w", err)
	}

	var rows []row
	if err := query().Select(selectWithAlias).
		Order("users.created_at DESC, users.id DESC").
		Limit(filter.Limit).Offset(filter.Offset).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("user : liste : %w", err)
	}

	users := make([]User, 0, len(rows))
	for _, found := range rows {
		u, err := found.toUser(r.sealer)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
	}
	return users, total, nil
}

func (r *gormRepository) ActiveEmailsByRole(ctx context.Context, roles ...Role) ([]string, error) {
	names := make([]string, len(roles))
	for i, role := range roles {
		names[i] = string(role)
	}
	var emails []string
	err := r.conn(ctx).Table(tableUsers).
		Where("role IN ? AND status = ?", names, string(StatusActive)).
		Order("email").
		Pluck("email", &emails).Error
	if err != nil {
		return nil, fmt.Errorf("user : adresses par rôle : %w", err)
	}
	return emails, nil
}

// escapeLike neutralise les caractères spéciaux de LIKE dans une saisie utilisateur.
func escapeLike(value string) string {
	return strings.NewReplacer(likeEscape, likeEscape+likeEscape, "%", likeEscape+"%", "_", likeEscape+"_").Replace(value)
}
