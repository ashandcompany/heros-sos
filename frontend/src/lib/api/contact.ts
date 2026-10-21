import type { ContactMessage, ContactMessageResult } from "@/types";
import { mockResponse } from "./config";

/**
 * Transmet un message du formulaire de contact à l'équipe du Bureau.
 * Future implémentation : `POST ${API_URL}/contact/messages` avec `{ name, email, subject, message }`.
 */
export async function sendContactMessage(message: ContactMessage): Promise<ContactMessageResult> {
  void message;
  return mockResponse({ ok: true });
}
