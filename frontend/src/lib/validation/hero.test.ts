import { describe, expect, it } from "vitest";
import {
  ALIAS_MAX_LENGTH,
  ALIAS_MIN_LENGTH,
  BIO_MAX_LENGTH,
  REJECTION_REASON_MAX_LENGTH,
  heroRequestFieldErrors,
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
