import type { ContactMessage, ContactMessageResult } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure, visitorHeaders } from "./http";

/**
 * Transmet un message du formulaire de contact à l'équipe du Bureau (par mail, rien n'est
 * conservé). API : `POST /contact/messages`, limité par adresse IP : celle du visiteur est
 * transmise (`visitorHeaders`).
 */
export async function sendContactMessage(message: ContactMessage): Promise<ContactMessageResult> {
  if (!isApiEnabled) return mockResponse({ ok: true });

  try {
    await apiFetch("/contact/messages", {
      method: "POST",
      body: {
        name: message.name,
        email: message.email,
        subject: message.subject,
        message: message.message,
      } satisfies ApiSchema<"ContactMessageRequest">,
      headers: await visitorHeaders(),
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "validation_failed") {
        return { ok: false, error: "invalid-fields", fields: error.fields };
      }
      if (error.code === "contact_delivery_unavailable") {
        return { ok: false, error: "delivery-unavailable" };
      }
    }
    return toApiFailure(error, "Message de contact");
  }
}
