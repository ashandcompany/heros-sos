import { NextResponse, type NextRequest } from "next/server";
import { SESSION_COOKIE } from "@/lib/cookies";
import { accountHomeByRole, routes } from "@/lib/routes";
import { getMfaRedirect, getUserFromToken } from "@/lib/session";
import type { UserRole } from "@/types";

/** Espaces réservés à un rôle (le préfixe couvre aussi les sous-pages). */
const protectedAreas: Array<{ prefix: string; role: UserRole }> = [
  { prefix: "/admin", role: "admin" },
  { prefix: "/profil", role: "citoyen" },
  { prefix: "/heros", role: "super-heros" },
];

/**
 * Contrôle d'accès « optimiste », avant le rendu :
 * - /admin/* et /profil/* : réservés à un rôle (visiteur → connexion, autre rôle → son espace) ;
 * - pages réservées aux visiteurs (mot de passe oublié) : un utilisateur connecté est
 *   redirigé vers son espace.
 */
export async function proxy(request: NextRequest) {
  const sessionToken = request.cookies.get(SESSION_COOKIE)?.value;
  const user = await getUserFromToken(sessionToken);
  const redirectTo = (path: string) => NextResponse.redirect(new URL(path, request.url));

  const { pathname } = request.nextUrl;
  const area = protectedAreas.find(
    ({ prefix }) => pathname === prefix || pathname.startsWith(`${prefix}/`),
  );

  if (area) {
    // Une session limitée (2FA pas encore validée) n'est pas un utilisateur : elle est envoyée à la 2FA.
    if (!user) return redirectTo((await getMfaRedirect(sessionToken)) ?? routes.login);
    if (user.role !== area.role) return redirectTo(accountHomeByRole[user.role]);
    return NextResponse.next();
  }

  if (user) return redirectTo(accountHomeByRole[user.role]);

  return NextResponse.next();
}

export const config = {
  // Valeurs littérales obligatoires (analysées au build). Voir `routes` dans src/lib/routes.ts.
  matcher: ["/mot-de-passe-oublie", "/admin/:path*", "/profil/:path*", "/heros/:path*"],
};
