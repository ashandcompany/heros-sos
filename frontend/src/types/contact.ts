export type ContactSubject = "question" | "compte" | "donnees" | "partenariat" | "autre";

export interface ContactMessage {
  name: string;
  email: string;
  subject: ContactSubject;
  message: string;
}

export interface ContactMessageResult {
  ok: boolean;
}
