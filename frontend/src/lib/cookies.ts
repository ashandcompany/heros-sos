export const NEWSLETTER_COOKIE = "newsletter_subscribed";
export const SESSION_COOKIE = "session";

/** Options du cookie de session. Placeholder : le format sera fixé avec l'API Go. */
export const sessionCookieOptions = {
  maxAge: 60 * 60 * 24 * 7,
  path: "/",
  sameSite: "lax",
  httpOnly: true,
  secure: process.env.NODE_ENV === "production",
} as const;
