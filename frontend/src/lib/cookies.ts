export const NEWSLETTER_COOKIE = "newsletter_subscribed";

/** Cookies Secure (HTTPS) en production : même règle que le backend (`HEROS_COOKIE_SECURE`). */
const secureCookies = process.env.NODE_ENV === "production";

/**
 * Nom du cookie de session, identique à celui de l'API Go : `__Host-session` quand le cookie est
 * Secure (le préfixe impose Secure, `Path=/` et l'absence de `Domain`), `session` sinon.
 */
export const SESSION_COOKIE = secureCookies ? "__Host-session" : "session";

/**
 * Attributs du cookie de session, alignés sur ceux de l'API. La durée n'en fait pas partie :
 * elle vient de l'API (`expiresAt` du login), ou de `MOCK_SESSION_MAX_AGE` pour les mocks.
 */
export const sessionCookieAttributes = {
  path: "/",
  sameSite: "lax",
  httpOnly: true,
  secure: secureCookies,
} as const;

/** Durée d'une session mock (7 jours), sans API pour la fixer. */
export const MOCK_SESSION_MAX_AGE = 60 * 60 * 24 * 7;
