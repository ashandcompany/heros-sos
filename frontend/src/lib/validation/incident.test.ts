import { describe, expect, it } from "vitest";
import {
  DESCRIPTION_MAX_LENGTH,
  DESCRIPTION_MIN_LENGTH,
  TITLE_MAX_LENGTH,
  isKnownTypeId,
  toDeclaration,
  validateDeclaration,
  type DeclarationValues,
} from "./incident";

const values: DeclarationValues = {
  reporterRole: "temoin",
  type: "3",
  title: "Braquage en cours",
  description: "Un individu masqué menace les clients de la banque.",
  address: "1 place de la Bourse",
  city: "Lyon",
  occurredAt: "",
  villainId: "",
};
const options = { villainIds: ["villain-1"], typeIds: [1, 3] };

const hasError = (errors: Record<string, string | undefined>) =>
  Object.values(errors).some(Boolean);

describe("validateDeclaration", () => {
  it("accepte une déclaration complète", () => {
    expect(hasError(validateDeclaration(values, options))).toBe(false);
  });

  it("applique les bornes du titre et de la description", () => {
    const at = (title: string, description: string) =>
      validateDeclaration({ ...values, title, description }, options);

    expect(at("a".repeat(TITLE_MAX_LENGTH), "d".repeat(DESCRIPTION_MAX_LENGTH))).toMatchObject({
      title: undefined,
      description: undefined,
    });
    expect(at("a".repeat(TITLE_MAX_LENGTH + 1), values.description).title).toBeDefined();
    expect(at("abcd", values.description).title).toBeDefined();
    expect(at(values.title, "d".repeat(DESCRIPTION_MIN_LENGTH - 1)).description).toBeDefined();
    expect(at(values.title, "d".repeat(DESCRIPTION_MAX_LENGTH + 1)).description).toBeDefined();
  });

  it("demande si le déclarant est témoin ou victime", () => {
    const withoutRole = { ...values, reporterRole: "" };
    expect(validateDeclaration(withoutRole, options).reporterRole).toBeDefined();
  });

  it("refuse un type, un vilain ou une date inconnus", () => {
    const tomorrow = new Date(Date.now() + 86_400_000).toISOString();
    const errors = validateDeclaration(
      { ...values, type: "2", villainId: "villain-9", occurredAt: tomorrow },
      options,
    );
    expect(errors.type).toBeDefined();
    expect(errors.villainId).toBeDefined();
    expect(errors.occurredAt).toBeDefined();
  });
});

describe("isKnownTypeId", () => {
  it("n'accepte qu'un identifiant entier du référentiel", () => {
    expect(isKnownTypeId("3", [3])).toBe(true);
    expect(isKnownTypeId("3.0", [3])).toBe(false);
    expect(isKnownTypeId("", [3])).toBe(false);
    expect(isKnownTypeId("4", [3])).toBe(false);
  });
});

describe("toDeclaration", () => {
  it("convertit le type en nombre et les champs vides en null", () => {
    expect(toDeclaration({ ...values, reporterRole: "" })).toEqual({
      reporterRole: null,
      typeId: 3,
      title: values.title,
      description: values.description,
      location: { address: values.address, city: values.city },
      occurredAt: null,
      villainId: null,
    });
  });

  it("transmet la date de l'incident en ISO", () => {
    const occurredAt = "2026-10-21T08:30:00.000Z";
    expect(toDeclaration({ ...values, occurredAt }).occurredAt).toBe(occurredAt);
  });
});
