import type { NewsletterSubscriptionResult } from "@/types";
import { mockResponse } from "./config";

/**
 * Inscrit une adresse email à la newsletter.
 * Future implémentation : `POST ${API_URL}/newsletter/subscriptions` avec `{ email }`.
 */
export async function subscribeToNewsletter(email: string): Promise<NewsletterSubscriptionResult> {
  void email;
  return mockResponse({ ok: true });
}
