package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/user"
)

// Principal est l'utilisateur authentifié d'une requête. C'est le contrat d'identité du
// backend : tous les handlers lisent l'utilisateur courant par PrincipalFrom, sans connaître la
// façon dont la session est stockée. Les tests des autres domaines l'injectent avec WithPrincipal.
type Principal struct {
	UserID uuid.UUID
	Role   user.Role
	// MFAVerified indique que la double authentification a été validée pour cette session.
	MFAVerified bool
}

// IsModerator indique si l'utilisateur est modérateur.
func (p Principal) IsModerator() bool { return p.Role == user.RoleModerator }

type principalKey struct{}

// WithPrincipal retourne un contexte portant l'utilisateur authentifié.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

// PrincipalFrom retourne l'utilisateur authentifié du contexte, s'il y en a un.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}
