// Package bootstrap crée les comptes qui ne peuvent pas passer par l'inscription publique : le
// premier modérateur, qui valide ensuite les super-héros et les autres comptes.
package bootstrap

import (
	"context"
	"fmt"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/ids"
	"github.com/ashandcompany/heros-sos/backend/internal/user"
	"github.com/ashandcompany/heros-sos/backend/internal/validation"
)

// ModeratorDeps regroupe les dépendances de la création d'un modérateur.
type ModeratorDeps struct {
	Users    user.Repository
	Hasher   *auth.PasswordHasher
	Clock    clock.Clock
	Password config.PasswordPolicy
}

// ModeratorInput est l'identité du modérateur. Ni téléphone ni adresse : ce compte de service
// n'en a pas besoin.
type ModeratorInput struct {
	Email     string
	FirstName string
	LastName  string
	BirthDate string
	Password  string
}

// CreateModerator crée un compte modérateur déjà actif, avec la même validation et le même
// stockage (mot de passe haché, données chiffrées) qu'une inscription. Une adresse déjà
// utilisée retourne user.ErrEmailTaken.
func CreateModerator(ctx context.Context, deps ModeratorDeps, in ModeratorInput) (*user.User, error) {
	now := deps.Clock.Now()
	var fields validation.Collector
	account := &user.User{
		ID: ids.New(), Role: user.RoleModerator, Status: user.StatusPendingActivation,
		Civility: validation.CivilityUnspecific,
	}

	var code string
	account.Email, code = validation.Email(in.Email)
	fields.Add("email", code)
	account.FirstName, code = validation.Name(in.FirstName)
	fields.Add("firstName", code)
	account.LastName, code = validation.Name(in.LastName)
	fields.Add("lastName", code)
	account.BirthDate, code = validation.BirthDate(in.BirthDate, now)
	fields.Add("birthDate", code)
	fields.Add("password", validation.Password(in.Password, validation.PasswordPolicy{
		MinLength: deps.Password.MinLength, MaxLength: deps.Password.MaxLength,
	}))
	if err := fields.Err(); err != nil {
		return nil, err
	}

	hash, err := deps.Hasher.Hash(ctx, in.Password)
	if err != nil {
		return nil, fmt.Errorf("bootstrap : hachage du mot de passe : %w", err)
	}
	account.PasswordHash = hash

	if err := deps.Users.Create(ctx, account); err != nil {
		return nil, err
	}
	if _, err := deps.Users.Activate(ctx, account.ID, now); err != nil {
		return nil, err
	}
	account.Status = user.StatusActive
	return account, nil
}
