import { describe, expect, it } from "vitest";
import {
  MESSAGE_MAX_LENGTH,
  MESSAGE_MIN_LENGTH,
  NAME_MAX_LENGTH,
  contactMessageFailureState,
  validateContact,
  type ContactValues,
} from "./contact";

const values: ContactValues = {
  name: "Camille Dupont",
  email: "camille@exemple.fr",
  subject: "question",
  message: "Comment devenir super-héros sur la plateforme ?",
};

describe("validateContact", () => {
  it("accepte un message valide", () => {
    expect(Object.values(validateContact(values)).some(Boolean)).toBe(false);
  });

  it("applique les bornes du nom et du message", () => {
    expect(
      validateContact({ ...values, name: "a".repeat(NAME_MAX_LENGTH + 1) }).name,
    ).toBeDefined();
    expect(
      validateContact({ ...values, message: "a".repeat(MESSAGE_MIN_LENGTH - 1) }).message,
    ).toBeDefined();
    expect(
      validateContact({ ...values, message: "a".repeat(MESSAGE_MAX_LENGTH + 1) }).message,
    ).toBeDefined();
  });

  it("refuse un objet inconnu et un email invalide", () => {
    const errors = validateContact({ ...values, subject: "spam", email: "camille@" });
    expect(errors.subject).toBeDefined();
    expect(errors.email).toBeDefined();
  });
});

describe("contactMessageFailureState", () => {
  it("place les erreurs de l'API sur les champs connus", () => {
    const state = contactMessageFailureState(
      { ok: false, error: "invalid-fields", fields: { email: "invalid" } },
      values,
    );
    expect(state.fieldErrors.email).toBeDefined();
    expect(state.message).toBeUndefined();
  });

  it("affiche un message global si aucun champ refusé n'est dans le formulaire", () => {
    const state = contactMessageFailureState(
      { ok: false, error: "invalid-fields", fields: { other: "invalid" } },
      values,
    );
    expect(state.fieldErrors).toEqual({});
    expect(state.message).toBeDefined();
    expect(state.values).toEqual(values);
  });
});
