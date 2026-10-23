import { describe, expect, it } from "vitest";
import {
  ALIAS_MAX_LENGTH,
  ALIAS_MIN_LENGTH,
  BIO_MAX_LENGTH,
  REJECTION_REASON_MAX_LENGTH,
  heroPublicProfileFieldErrors,
  heroRequestFieldErrors,
  readHeroPublicProfileValues,
  validateHeroPublicProfile,
  validateHeroRequest,
  validateRejectionReason,
} from "./hero";

const request = { alias: "Lumen", bio: "", requestMessage: "" };

describe("validateHeroRequest", () => {
  it.each([
    ["Lumen", true],
    ["Él-Ombre 2", true],
    ["Super_Nova", true],
    ["a".repeat(ALIAS_MIN_LENGTH - 1), false],
    ["a".repeat(ALIAS_MAX_LENGTH + 1), false],
    ["Lumen!", false],
  ])("pseudo %s valide : %s", (alias, valid) => {
    expect(validateHeroRequest({ ...request, alias }).alias === undefined).toBe(valid);
  });

  it("borne la description", () => {
    expect(
      validateHeroRequest({ ...request, bio: "a".repeat(BIO_MAX_LENGTH) }).bio,
    ).toBeUndefined();
    expect(
      validateHeroRequest({ ...request, bio: "a".repeat(BIO_MAX_LENGTH + 1) }).bio,
    ).toBeDefined();
  });
});

describe("heroRequestFieldErrors", () => {
  it("distingue un pseudo manquant d'un pseudo invalide", () => {
    const missing = heroRequestFieldErrors({ alias: "required" }).alias;
    const invalid = heroRequestFieldErrors({ alias: "invalid" }).alias;
    expect(missing).toBeDefined();
    expect(invalid).toBeDefined();
    expect(missing).not.toBe(invalid);
  });
});

describe("validateRejectionReason", () => {
  it("exige un motif borné", () => {
    expect(validateRejectionReason("")).toBeDefined();
    expect(validateRejectionReason("Dossier incomplet")).toBeUndefined();
    expect(validateRejectionReason("a".repeat(REJECTION_REASON_MAX_LENGTH + 1))).toBeDefined();
  });
});

describe("validateHeroPublicProfile", () => {
  it("applique les règles du pseudo et de la description", () => {
    expect(validateHeroPublicProfile({ alias: "Lumen", power: "" })).toEqual({
      alias: undefined,
      power: undefined,
    });
    const errors = validateHeroPublicProfile({
      alias: "L!",
      power: "a".repeat(BIO_MAX_LENGTH + 1),
    });
    expect(errors.alias).toBeDefined();
    expect(errors.power).toBeDefined();
  });

  it("traduit les champs refusés par l'API vers ceux du formulaire", () => {
    const errors = heroPublicProfileFieldErrors({ alias: "invalid", bio: "too_long" });
    expect(errors.alias).toBeDefined();
    expect(errors.power).toBeDefined();
  });

  it("lit et nettoie le formulaire", () => {
    const formData = new FormData();
    formData.set("alias", " Lumen ");
    expect(readHeroPublicProfileValues(formData)).toEqual({ alias: "Lumen", power: "" });
  });
});
