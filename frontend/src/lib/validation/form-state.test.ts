import { describe, expect, it } from "vitest";
import { apiFailureMessage, apiFieldErrors, hasFieldErrors } from "./form-state";

describe("apiFieldErrors", () => {
  it("retire le préfixe `address.` et ignore les champs absents du formulaire", () => {
    const errors = apiFieldErrors(
      { "address.city": "required", email: "too_long", password: "weak" },
      ["city", "email"] as const,
    );
    expect(errors).toEqual({
      city: "Ce champ est obligatoire.",
      email: "Ce champ est trop long.",
    });
  });

  it("donne un message générique pour un code inconnu", () => {
    expect(apiFieldErrors({ email: "weird" }, ["email"] as const).email).toBe(
      "Cette valeur n'est pas valide.",
    );
  });
});

describe("hasFieldErrors", () => {
  it("ignore les champs validés sans erreur", () => {
    expect(hasFieldErrors({ email: undefined, city: "" })).toBe(false);
    expect(hasFieldErrors({ email: undefined, city: "Indiquez la ville." })).toBe(true);
  });
});

describe("apiFailureMessage", () => {
  it("annonce le délai d'attente d'un 429", () => {
    expect(
      apiFailureMessage({ ok: false, error: "too-many-attempts", retryAfterSeconds: 90 }),
    ).toContain("2 minutes");
  });

  it.each(["forbidden", "payload-too-large", "unsupported-media-type", "unavailable"] as const)(
    "a un message pour %s",
    (error) => {
      expect(apiFailureMessage({ ok: false, error })).toBeTruthy();
    },
  );
});
