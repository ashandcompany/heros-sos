package user_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/crypto"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

var baseTime = time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)

type fixture struct {
	db   *gorm.DB
	repo user.Repository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	sealer, err := crypto.NewSealer(map[uint16][]byte{1: bytes.Repeat([]byte{0x42}, crypto.MasterKeyLength)}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{db: db, repo: user.NewRepository(db, sealer)}
}

func newUser(email string) *user.User {
	return &user.User{
		ID:           ids.New(),
		Email:        email,
		PasswordHash: "$argon2id$hash",
		Role:         user.RoleCitizen,
		Status:       user.StatusPendingActivation,
		Civility:     "monsieur",
		FirstName:    "Steve",
		LastName:     "Rogers",
		BirthDate:    time.Date(1918, time.July, 4, 0, 0, 0, 0, time.UTC),
		Phone:        "0612345678",
		Address: user.Address{
			Street:     "569 Leaman Place",
			Complement: "Bâtiment B",
			PostalCode: "11201",
			City:       "Brooklyn",
			Country:    "US",
		},
	}
}

func (f fixture) create(t *testing.T, u *user.User) *user.User {
	t.Helper()
	if err := f.repo.Create(context.Background(), u); err != nil {
		t.Fatalf("Create(%s) : %v", u.Email, err)
	}
	return u
}

func TestCreateAndFindDecryptsPersonalData(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, newUser("steve@avengers.example"))

	for name, find := range map[string]func() (*user.User, error){
		"par identifiant": func() (*user.User, error) { return f.repo.FindByID(context.Background(), created.ID) },
		"par email":       func() (*user.User, error) { return f.repo.FindByEmail(context.Background(), "steve@avengers.example") },
	} {
		t.Run(name, func(t *testing.T) {
			got, err := find()
			if err != nil {
				t.Fatal(err)
			}
			if got.FirstName != "Steve" || got.LastName != "Rogers" || got.Phone != "0612345678" {
				t.Errorf("identité = %q %q %q", got.FirstName, got.LastName, got.Phone)
			}
			if got.Address != created.Address {
				t.Errorf("adresse = %+v, attendu %+v", got.Address, created.Address)
			}
			if !got.BirthDate.Equal(created.BirthDate) {
				t.Errorf("date de naissance = %v", got.BirthDate)
			}
			if got.Role != user.RoleCitizen || got.Status != user.StatusPendingActivation {
				t.Errorf("rôle = %q, statut = %q", got.Role, got.Status)
			}
			if got.DisplayName() != "Steve" {
				t.Errorf("nom affiché = %q", got.DisplayName())
			}
		})
	}
}

func TestPersonalDataIsEncryptedInDatabase(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, newUser("steve@avengers.example"))

	var raw struct {
		LastName, FirstName, BirthDate, Phone, Street, City []byte
	}
	if err := f.db.Raw(`SELECT last_name, first_name, birth_date, phone, street, city FROM users WHERE id = ?`, created.ID).
		Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}

	for name, stored := range map[string][]byte{
		"last_name": raw.LastName, "first_name": raw.FirstName, "birth_date": raw.BirthDate,
		"phone": raw.Phone, "street": raw.Street, "city": raw.City,
	} {
		for _, secret := range []string{"Rogers", "Steve", "1918", "0612345678", "Leaman", "Brooklyn"} {
			if bytes.Contains(stored, []byte(secret)) {
				t.Errorf("la colonne %s contient %q en clair", name, secret)
			}
		}
		if len(stored) < crypto.Overhead {
			t.Errorf("la colonne %s ne ressemble pas à un chiffré (%d octets)", name, len(stored))
		}
	}
}

func TestEmptyOptionalFieldsAreStoredAsNull(t *testing.T) {
	f := newFixture(t)
	u := newUser("sans-complement@example.org")
	u.Address.Complement = ""
	f.create(t, u)

	var stored struct{ IsNull bool }
	if err := f.db.Raw(`SELECT address_complement IS NULL AS is_null FROM users WHERE id = ?`, u.ID).Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if !stored.IsNull {
		t.Error("un complément d'adresse vide doit être stocké NULL")
	}
	got, _ := f.repo.FindByID(context.Background(), u.ID)
	if got.Address.Complement != "" {
		t.Errorf("complément = %q", got.Address.Complement)
	}
}

func TestCiphertextCannotBeMovedToAnotherRow(t *testing.T) {
	f := newFixture(t)
	victim := f.create(t, newUser("victime@example.org"))
	attacker := f.create(t, newUser("attaquant@example.org"))

	// Un attaquant qui peut écrire dans la base copie le nom chiffré d'une ligne vers une autre.
	err := f.db.Exec(`UPDATE users SET last_name = (SELECT last_name FROM users WHERE id = ?) WHERE id = ?`,
		victim.ID, attacker.ID).Error
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.repo.FindByID(context.Background(), attacker.ID)
	if !errors.Is(err, crypto.ErrDecryption) {
		t.Fatalf("erreur = %v, attendu ErrDecryption : le chiffré est lié à sa ligne", err)
	}
}

func TestEmailIsUniqueIgnoringCase(t *testing.T) {
	f := newFixture(t)
	f.create(t, newUser("tony@stark.io"))

	err := f.repo.Create(context.Background(), newUser("TONY@Stark.IO"))
	if !errors.Is(err, user.ErrEmailTaken) {
		t.Fatalf("erreur = %v, attendu ErrEmailTaken", err)
	}

	got, err := f.repo.FindByEmail(context.Background(), "Tony@STARK.io")
	if err != nil || got.Email != "tony@stark.io" {
		t.Fatalf("la recherche par email doit ignorer la casse : (%v, %v)", got, err)
	}
}

func TestConcurrentRegistrationsOfSameEmailYieldOneAccount(t *testing.T) {
	f := newFixture(t)

	const attempts = 16
	var wg sync.WaitGroup
	results := make(chan error, attempts)
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- f.repo.Create(context.Background(), newUser("course@example.org"))
		}()
	}
	wg.Wait()
	close(results)

	var created, taken int
	for err := range results {
		switch {
		case err == nil:
			created++
		case errors.Is(err, user.ErrEmailTaken):
			taken++
		default:
			t.Errorf("erreur inattendue : %v", err)
		}
	}
	if created != 1 || taken != attempts-1 {
		t.Fatalf("créés = %d, refusés = %d ; attendu 1 et %d", created, taken, attempts-1)
	}
}

func TestFindNotFound(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if _, err := f.repo.FindByID(ctx, uuid.New()); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("FindByID : %v", err)
	}
	if _, err := f.repo.FindByEmail(ctx, "personne@example.org"); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("FindByEmail : %v", err)
	}
	if _, err := f.repo.FindHeroByAlias(ctx, "Inconnu"); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("FindHeroByAlias : %v", err)
	}
}

func TestFindHeroByAlias(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	hero := newUser("tony@stark.io")
	hero.Role = user.RoleHero
	f.create(t, hero)
	citizen := newUser("pepper@stark.io")
	f.create(t, citizen)

	for _, profile := range []struct {
		id    uuid.UUID
		alias string
	}{{hero.ID, "Iron Man"}, {citizen.ID, "Faux Héros"}} {
		if err := f.db.Exec(`INSERT INTO hero_profiles (user_id, alias) VALUES (?, ?)`, profile.id, profile.alias).Error; err != nil {
			t.Fatal(err)
		}
	}

	got, err := f.repo.FindHeroByAlias(ctx, "iron man")
	if err != nil {
		t.Fatalf("la recherche par pseudo doit ignorer la casse : %v", err)
	}
	if got.ID != hero.ID || got.Alias != "Iron Man" || got.DisplayName() != "Iron Man" {
		t.Errorf("héros = %+v", got)
	}

	// Un pseudo demandé mais pas encore validé (rôle citoyen) ne permet pas de se connecter.
	if _, err := f.repo.FindHeroByAlias(ctx, "Faux Héros"); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("un citoyen ne doit pas être trouvé par pseudo : %v", err)
	}
}

func TestActivateOnlyOnce(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	u := f.create(t, newUser("activation@example.org"))

	changed, err := f.repo.Activate(ctx, u.ID, baseTime)
	if err != nil || !changed {
		t.Fatalf("première activation = (%v, %v)", changed, err)
	}
	changed, err = f.repo.Activate(ctx, u.ID, baseTime)
	if err != nil || changed {
		t.Fatalf("seconde activation = (%v, %v), attendu (false, nil)", changed, err)
	}

	got, _ := f.repo.FindByID(ctx, u.ID)
	if got.Status != user.StatusActive || got.ActivatedAt == nil || !got.ActivatedAt.Equal(baseTime) {
		t.Errorf("compte = %+v", got)
	}
}

func TestLoginFailureLockoutGrowsAndIsCapped(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	u := f.create(t, newUser("verrou@example.org"))
	lockout := config.Lockout{MaxFailures: 3, BaseDelay: time.Minute, MaxDelay: 10 * time.Minute}

	tests := []struct {
		failure   int
		wantDelay time.Duration // 0 = pas de verrouillage
	}{
		{1, 0}, {2, 0},
		{3, time.Minute}, {4, 2 * time.Minute}, {5, 4 * time.Minute}, {6, 8 * time.Minute},
		{7, 10 * time.Minute}, {8, 10 * time.Minute}, // plafonné
	}
	for _, test := range tests {
		result, err := f.repo.RecordLoginFailure(ctx, u.ID, baseTime, lockout)
		if err != nil {
			t.Fatalf("échec %d : %v", test.failure, err)
		}
		if result.Count != test.failure {
			t.Errorf("échec %d : compteur = %d", test.failure, result.Count)
		}
		switch {
		case test.wantDelay == 0 && result.LockedUntil != nil:
			t.Errorf("échec %d : verrouillé trop tôt (%v)", test.failure, result.LockedUntil)
		case test.wantDelay > 0 && (result.LockedUntil == nil || !result.LockedUntil.Equal(baseTime.Add(test.wantDelay))):
			t.Errorf("échec %d : verrouillé jusqu'à %v, attendu %v", test.failure, result.LockedUntil, baseTime.Add(test.wantDelay))
		}
	}

	got, _ := f.repo.FindByID(ctx, u.ID)
	if !got.IsLocked(baseTime.Add(9*time.Minute)) || got.IsLocked(baseTime.Add(11*time.Minute)) {
		t.Errorf("IsLocked incohérent pour verrou jusqu'à %v", got.LockedUntil)
	}
}

func TestConcurrentLoginFailuresAreAllCounted(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	u := f.create(t, newUser("brute-force@example.org"))
	lockout := config.Lockout{MaxFailures: 5, BaseDelay: time.Minute, MaxDelay: time.Hour}

	const attempts = 24
	var wg sync.WaitGroup
	errs := make(chan error, attempts)
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.repo.RecordLoginFailure(ctx, u.ID, baseTime, lockout)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	got, _ := f.repo.FindByID(ctx, u.ID)
	if got.FailedLoginCount != attempts {
		t.Fatalf("compteur = %d, attendu %d : des échecs simultanés ont été perdus", got.FailedLoginCount, attempts)
	}
}

func TestLoginSuccessAndPasswordChangeClearLockout(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	lockout := config.Lockout{MaxFailures: 1, BaseDelay: time.Hour, MaxDelay: time.Hour}

	locked := func(email string) *user.User {
		u := f.create(t, newUser(email))
		if _, err := f.repo.RecordLoginFailure(ctx, u.ID, baseTime, lockout); err != nil {
			t.Fatal(err)
		}
		return u
	}

	afterSuccess := locked("succes@example.org")
	if err := f.repo.RecordLoginSuccess(ctx, afterSuccess.ID, baseTime.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, _ := f.repo.FindByID(ctx, afterSuccess.ID)
	if got.FailedLoginCount != 0 || got.LockedUntil != nil || got.LastLoginAt == nil {
		t.Errorf("après succès : %+v", got)
	}

	afterReset := locked("reset@example.org")
	if err := f.repo.UpdatePasswordHash(ctx, afterReset.ID, "$argon2id$nouveau", baseTime.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, _ = f.repo.FindByID(ctx, afterReset.ID)
	if got.PasswordHash != "$argon2id$nouveau" || got.FailedLoginCount != 0 || got.LockedUntil != nil {
		t.Errorf("après changement de mot de passe : %+v", got)
	}

	if err := f.repo.UpdatePasswordHash(ctx, uuid.New(), "x", baseTime); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("compte inconnu : %v", err)
	}
	if _, err := f.repo.RecordLoginFailure(ctx, uuid.New(), baseTime, lockout); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("échec sur compte inconnu : %v", err)
	}
}

func TestDeactivateAndReactivateRestorePreviousState(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	active := f.create(t, newUser("actif@example.org"))
	if _, err := f.repo.Activate(ctx, active.ID, baseTime); err != nil {
		t.Fatal(err)
	}
	pending := f.create(t, newUser("attente@example.org"))

	tests := []struct {
		name string
		id   uuid.UUID
		want user.Status
	}{
		{"compte actif", active.ID, user.StatusActive},
		{"compte jamais activé", pending.ID, user.StatusPendingActivation},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed, err := f.repo.Deactivate(ctx, test.id, baseTime)
			if err != nil || !changed {
				t.Fatalf("Deactivate = (%v, %v)", changed, err)
			}
			if changed, _ = f.repo.Deactivate(ctx, test.id, baseTime); changed {
				t.Error("désactiver deux fois ne doit rien changer")
			}
			if got, _ := f.repo.FindByID(ctx, test.id); got.Status != user.StatusDeactivated || got.DeactivatedAt == nil {
				t.Fatalf("après désactivation : %+v", got)
			}

			changed, err = f.repo.Reactivate(ctx, test.id, baseTime)
			if err != nil || !changed {
				t.Fatalf("Reactivate = (%v, %v)", changed, err)
			}
			if got, _ := f.repo.FindByID(ctx, test.id); got.Status != test.want || got.DeactivatedAt != nil {
				t.Errorf("après réactivation : statut %q, attendu %q", got.Status, test.want)
			}
			if changed, _ = f.repo.Reactivate(ctx, test.id, baseTime); changed {
				t.Error("réactiver un compte non désactivé ne doit rien changer")
			}
		})
	}
}

func TestMarkVerifiedOnlyOnce(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	moderator := f.create(t, newUser("moderateur@example.org"))
	citizen := f.create(t, newUser("citoyen@example.org"))

	if changed, err := f.repo.MarkVerified(ctx, citizen.ID, moderator.ID, baseTime); err != nil || !changed {
		t.Fatalf("première vérification = (%v, %v)", changed, err)
	}
	if changed, _ := f.repo.MarkVerified(ctx, citizen.ID, moderator.ID, baseTime.Add(time.Hour)); changed {
		t.Error("la date de vérification ne doit pas être réécrite")
	}
	got, _ := f.repo.FindByID(ctx, citizen.ID)
	if got.VerifiedAt == nil || !got.VerifiedAt.Equal(baseTime) || got.VerifiedBy == nil || *got.VerifiedBy != moderator.ID {
		t.Errorf("vérification = %+v", got)
	}
}

func TestSetRole(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	u := f.create(t, newUser("role@example.org"))

	if err := f.repo.SetRole(ctx, u.ID, user.RoleHero, baseTime); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.repo.FindByID(ctx, u.ID); got.Role != user.RoleHero {
		t.Errorf("rôle = %q", got.Role)
	}
	if err := f.repo.SetRole(ctx, uuid.New(), user.RoleHero, baseTime); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("compte inconnu : %v", err)
	}
}

func TestListFiltersPaginatesAndEscapesPatterns(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	for i := range 5 {
		u := newUser(fmt.Sprintf("citoyen%d@example.org", i))
		f.create(t, u)
		if i%2 == 0 {
			if _, err := f.repo.Activate(ctx, u.ID, baseTime); err != nil {
				t.Fatal(err)
			}
		}
	}
	hero := newUser("heros@example.org")
	hero.Role = user.RoleHero
	f.create(t, hero)
	f.create(t, newUser("50%_off@example.org"))

	t.Run("sans filtre", func(t *testing.T) {
		users, total, err := f.repo.List(ctx, user.Filter{Limit: 100})
		if err != nil || total != 7 || len(users) != 7 {
			t.Fatalf("(%d utilisateurs, total %d, %v)", len(users), total, err)
		}
	})
	t.Run("par rôle", func(t *testing.T) {
		users, total, _ := f.repo.List(ctx, user.Filter{Role: user.RoleHero, Limit: 10})
		if total != 1 || len(users) != 1 || users[0].ID != hero.ID {
			t.Fatalf("(%d utilisateurs, total %d)", len(users), total)
		}
	})
	t.Run("par statut", func(t *testing.T) {
		_, total, _ := f.repo.List(ctx, user.Filter{Status: user.StatusActive, Limit: 10})
		if total != 3 {
			t.Fatalf("total = %d, attendu 3", total)
		}
	})
	t.Run("par préfixe d'email, sans tenir compte de la casse", func(t *testing.T) {
		_, total, _ := f.repo.List(ctx, user.Filter{EmailPrefix: "CITOYEN", Limit: 10})
		if total != 5 {
			t.Fatalf("total = %d, attendu 5", total)
		}
	})
	t.Run("les caractères spéciaux de LIKE sont littéraux", func(t *testing.T) {
		users, total, _ := f.repo.List(ctx, user.Filter{EmailPrefix: "50%_", Limit: 10})
		if total != 1 || len(users) != 1 {
			t.Fatalf("(%d utilisateurs, total %d)", len(users), total)
		}
		if _, total, _ = f.repo.List(ctx, user.Filter{EmailPrefix: "%", Limit: 10}); total != 0 {
			t.Fatalf("« %% » est un caractère littéral, pas un joker (total %d)", total)
		}
	})
	t.Run("pagination", func(t *testing.T) {
		first, total, _ := f.repo.List(ctx, user.Filter{Limit: 3})
		second, _, _ := f.repo.List(ctx, user.Filter{Limit: 3, Offset: 3})
		if total != 7 || len(first) != 3 || len(second) != 3 {
			t.Fatalf("pages = %d et %d, total %d", len(first), len(second), total)
		}
		for _, a := range first {
			for _, b := range second {
				if a.ID == b.ID {
					t.Fatalf("%s apparaît sur deux pages", a.Email)
				}
			}
		}
	})
}

func TestActiveEmailsByRole(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	create := func(email string, role user.Role, activate bool) {
		u := newUser(email)
		u.Role = role
		f.create(t, u)
		if activate {
			if _, err := f.repo.Activate(ctx, u.ID, baseTime); err != nil {
				t.Fatal(err)
			}
		}
	}
	create("b-moderateur@example.org", user.RoleModerator, true)
	create("a-moderateur@example.org", user.RoleModerator, true)
	create("inactif@example.org", user.RoleModerator, false)
	create("heros@example.org", user.RoleHero, true)
	create("citoyen@example.org", user.RoleCitizen, true)

	got, err := f.repo.ActiveEmailsByRole(ctx, user.RoleModerator, user.RoleHero)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a-moderateur@example.org", "b-moderateur@example.org", "heros@example.org"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("adresses = %v, attendu %v", got, want)
	}
}

func ptr[T any](value T) *T { return &value }

func TestUpdateContactReplacesOnlyWhatIsGiven(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	u := f.create(t, newUser("contact@example.org"))
	later := baseTime.Add(time.Hour)

	// Le téléphone seul : l'adresse (et son complément) reste intacte.
	if err := f.repo.UpdateContact(ctx, u.ID, user.ContactUpdate{Phone: ptr("0699887766")}, later); err != nil {
		t.Fatal(err)
	}
	got, err := f.repo.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phone != "0699887766" || got.Address != u.Address || !got.UpdatedAt.Equal(later) {
		t.Errorf("après le téléphone : %+v (mis à jour %v)", got, got.UpdatedAt)
	}

	// L'adresse sans complément : le complément actuel est conservé.
	address := &user.AddressUpdate{Street: "10 rue de Rivoli", PostalCode: "75004", City: "Paris", Country: "FR"}
	if err := f.repo.UpdateContact(ctx, u.ID, user.ContactUpdate{Address: address}, later); err != nil {
		t.Fatal(err)
	}
	got, _ = f.repo.FindByID(ctx, u.ID)
	want := user.Address{Street: "10 rue de Rivoli", Complement: "Bâtiment B", PostalCode: "75004", City: "Paris", Country: "FR"}
	if got.Address != want || got.Phone != "0699887766" {
		t.Errorf("après l'adresse : %+v", got)
	}

	// Un complément vide l'efface (stocké NULL, comme à la création).
	address.Complement = ptr("")
	if err := f.repo.UpdateContact(ctx, u.ID, user.ContactUpdate{Address: address}, later); err != nil {
		t.Fatal(err)
	}
	got, _ = f.repo.FindByID(ctx, u.ID)
	if got.Address.Complement != "" {
		t.Errorf("complément = %q", got.Address.Complement)
	}
	var stored struct{ IsNull bool }
	if err := f.db.Raw(`SELECT address_complement IS NULL AS is_null FROM users WHERE id = ?`, u.ID).Scan(&stored).Error; err != nil || !stored.IsNull {
		t.Errorf("complément effacé non NULL : %v %v", stored, err)
	}

	// Un complément renseigné le remplace.
	address.Complement = ptr("3e étage")
	if err := f.repo.UpdateContact(ctx, u.ID, user.ContactUpdate{Address: address}, later); err != nil {
		t.Fatal(err)
	}
	if got, _ = f.repo.FindByID(ctx, u.ID); got.Address.Complement != "3e étage" {
		t.Errorf("complément = %q", got.Address.Complement)
	}
}

func TestUpdateContactKeepsDataEncryptedAndBoundToTheRow(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	u := f.create(t, newUser("chiffre@example.org"))
	other := f.create(t, newUser("autre@example.org"))

	update := user.ContactUpdate{
		Phone:   ptr("0699887766"),
		Address: &user.AddressUpdate{Street: "10 rue de Rivoli", PostalCode: "75004", City: "Paris", Country: "FR"},
	}
	if err := f.repo.UpdateContact(ctx, u.ID, update, baseTime); err != nil {
		t.Fatal(err)
	}

	var raw struct{ Phone, Street, City []byte }
	if err := f.db.Raw(`SELECT phone, street, city FROM users WHERE id = ?`, u.ID).Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	for name, stored := range map[string][]byte{"phone": raw.Phone, "street": raw.Street, "city": raw.City} {
		for _, secret := range []string{"0699887766", "Rivoli", "Paris"} {
			if bytes.Contains(stored, []byte(secret)) {
				t.Errorf("la colonne %s contient %q en clair", name, secret)
			}
		}
	}

	// Le chiffré d'une ligne ne se lit pas sur une autre : le contexte authentifié lie la valeur à son compte.
	if err := f.db.Exec(`UPDATE users SET phone = ? WHERE id = ?`, raw.Phone, other.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.FindByID(ctx, other.ID); err == nil {
		t.Error("un chiffré déplacé vers un autre compte ne doit pas être lisible")
	}
}

func TestUpdateContactUnknownUser(t *testing.T) {
	f := newFixture(t)
	err := f.repo.UpdateContact(context.Background(), uuid.New(), user.ContactUpdate{Phone: ptr("0612345678")}, baseTime)
	if !errors.Is(err, user.ErrNotFound) {
		t.Errorf("erreur = %v", err)
	}
}

func TestChangeEmail(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	u := f.create(t, newUser("ancien@example.org"))
	other := f.create(t, newUser("occupee@example.org"))

	if err := f.repo.ChangeEmail(ctx, u.ID, "nouveau@example.org", baseTime); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.FindByEmail(ctx, "ancien@example.org"); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("l'ancienne adresse ne doit plus désigner le compte : %v", err)
	}
	if got, err := f.repo.FindByEmail(ctx, "NOUVEAU@example.org"); err != nil || got.ID != u.ID {
		t.Errorf("la nouvelle adresse (casse ignorée) doit désigner le compte : %v %v", got, err)
	}

	// Une adresse déjà prise, quelle que soit sa casse, est refusée par la base.
	if err := f.repo.ChangeEmail(ctx, u.ID, "OCCUPEE@example.org", baseTime); !errors.Is(err, user.ErrEmailTaken) {
		t.Errorf("adresse prise : %v", err)
	}
	if got, _ := f.repo.FindByID(ctx, other.ID); got.Email != "occupee@example.org" {
		t.Errorf("l'autre compte a été modifié : %q", got.Email)
	}
	if err := f.repo.ChangeEmail(ctx, uuid.New(), "personne@example.org", baseTime); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("compte inconnu : %v", err)
	}
}

func TestSetAvatarPathReturnsThePreviousOne(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	u := f.create(t, newUser("photo@example.org"))

	previous, err := f.repo.SetAvatarPath(ctx, u.ID, "premiere.jpg", baseTime)
	if err != nil || previous != "" {
		t.Fatalf("première photo : %q %v", previous, err)
	}
	previous, err = f.repo.SetAvatarPath(ctx, u.ID, "seconde.jpg", baseTime)
	if err != nil || previous != "premiere.jpg" {
		t.Fatalf("seconde photo : %q %v", previous, err)
	}
	if got, _ := f.repo.FindByID(ctx, u.ID); got.AvatarPath != "seconde.jpg" {
		t.Errorf("chemin = %q", got.AvatarPath)
	}
	if _, err := f.repo.SetAvatarPath(ctx, uuid.New(), "x.jpg", baseTime); !errors.Is(err, user.ErrNotFound) {
		t.Errorf("compte inconnu : %v", err)
	}
}

// Deux remplacements simultanés doivent chacun recevoir un « ancien » chemin différent : sinon une
// photo resterait sur le disque sans personne pour la supprimer.
func TestConcurrentAvatarReplacementsLeaveNoOrphan(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	u := f.create(t, newUser("course@example.org"))
	if _, err := f.repo.SetAvatarPath(ctx, u.ID, "depart.jpg", baseTime); err != nil {
		t.Fatal(err)
	}

	const writers = 8
	var wg sync.WaitGroup
	previous := make([]string, writers)
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := f.repo.SetAvatarPath(ctx, u.ID, fmt.Sprintf("photo-%d.jpg", i), baseTime)
			if err != nil {
				t.Error(err)
			}
			previous[i] = got
		}()
	}
	wg.Wait()

	// Chaque fichier écrit (« depart » et les huit nouveaux) sauf le dernier est l'ancien de quelqu'un.
	seen := make(map[string]bool)
	for _, path := range previous {
		if seen[path] {
			t.Fatalf("l'ancien chemin %q a été retourné deux fois : %v", path, previous)
		}
		seen[path] = true
	}
	final, _ := f.repo.FindByID(ctx, u.ID)
	if seen[final.AvatarPath] {
		t.Errorf("la photo finale %q a aussi été retournée comme ancienne", final.AvatarPath)
	}
}
