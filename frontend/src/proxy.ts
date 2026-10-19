import { NextResponse, type NextRequest } from "next/server";
import { SESSION_COOKIE } from "@/lib/cookies";
import { getAccountHome } from "@/lib/session";

/**
 * Pages réservées aux visiteurs non connectés : un utilisateur connecté est redirigé
 * vers son espace (profil citoyen ou espace héros).
 */
export async function proxy(request: NextRequest) {
  const accountHome = await getAccountHome(request.cookies.get(SESSION_COOKIE)?.value);

  if (accountHome) {
    return NextResponse.redirect(new URL(accountHome, request.url));
  }

  return NextResponse.next();
}

export const config = {
  // Doit rester une valeur littérale (analysée au build). Voir `routes.forgotPassword`.
  matcher: ["/mot-de-passe-oublie"],
};
