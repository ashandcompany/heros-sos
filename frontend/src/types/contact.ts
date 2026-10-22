import type { ApiFailure } from "./api";

export type ContactSubject = "question" | "compte" | "donnees" | "partenariat" | "autre";

export interface ContactMessage {
  name: string;
  email: string;
  subject: ContactSubject;
  message: string;
}

export type ContactMessageResult =
  | { ok: true }
  /** Champs refusés par l'API (422) : champ (`name`, `email`, `subject`, `message`) → code. */
  | { ok: false; error: "invalid-fields"; fields: Record<string, string> }
  /** Le message n'a pas pu être confié au service d'envoi (503) : il n'est pas parti. */
  | { ok: false; error: "delivery-unavailable" }
  | ApiFailure;
