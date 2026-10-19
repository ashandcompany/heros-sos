import { NextResponse, type NextRequest } from "next/server";
import { SESSION_COOKIE } from "@/lib/cookies";
import { accountHomeByRole, routes } from "@/lib/routes";
import { getUserFromToken } from "@/lib/session";

const ADMIN_PREFIX = "/admin";

/**
 * Contrôle d'accès « optimiste », avant le rendu :
 * - /admin/* : réservé aux administrateurs (visiteur → connexion, autre rôle → son espace) ;
 * - pages réservées aux visiteurs (mot de passe oublié) : un utilisateur connecté est
 *   redirigé vers son espace.
 */
export async function proxy(request: NextRequest) {
  const user = await getUserFromToken(request.cookies.get(SESSION_COOKIE)?.value);
  const redirectTo = (path: string) => NextResponse.redirect(new URL(path, request.url));

  if (request.nextUrl.pathname.startsWith(ADMIN_PREFIX)) {
    if (!user) return redirectTo(routes.login);
    if (user.role !== "admin") return redirectTo(accountHomeByRole[user.role]);
    return NextResponse.next();
  }

  if (user) return redirectTo(accountHomeByRole[user.role]);

  return NextResponse.next();
}

export const config = {
  // Valeurs littérales obligatoires (analysées au build). Voir `routes` dans src/lib/routes.ts.
  matcher: ["/mot-de-passe-oublie", "/admin/:path*"],
};
