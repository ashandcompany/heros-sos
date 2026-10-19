import { cookies } from "next/headers";
import { NEWSLETTER_COOKIE } from "@/lib/cookies";
import { NewsletterForm } from "./NewsletterForm";

/** Lit le cookie d'inscription côté serveur (doit être rendu dans un <Suspense>). */
export async function NewsletterSection() {
  const alreadySubscribed = (await cookies()).has(NEWSLETTER_COOKIE);
  return <NewsletterForm alreadySubscribed={alreadySubscribed} />;
}
