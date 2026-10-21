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
	// MFAVerified indique que la session est complète : la double authentification a été validée, ou elle
	// n'est ni activée ni exigée pour ce compte. Une session qui ne l'est pas (code à saisir, ou activation
	// obligatoire pas encore faite) ne donne accès qu'à la 2FA et à la déconnexion (voir Protected).
	MFAVerified bool
	// MFAEnrolled indique que le compte a activé la 2FA.
	MFAEnrolled bool
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
