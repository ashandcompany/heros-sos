import { describe, expect, it } from "vitest";
import {
  PASSWORD_MIN_LENGTH,
  emptySignupValues,
  firstInvalidField,
  hasErrors,
  readSignupValues,
  stepOf,
  toSignupPayload,
  validateNewPassword,
  validateSignup,
  type SignupValues,
} from "./signup";

const validValues: SignupValues = {
  ...emptySignupValues,
  civility: "madame",
  firstName: "Camille",
  lastName: "Dupont-Lefèvre",
  birthDate: "1990-04-12",
  email: "Camille@Exemple.fr",
  mobilePhone: "06 12 34 56 78",
  street: "12 rue de la Paix",
  postalCode: "75002",
  city: "Paris",
  country: "FR",
  password: "Motdepasse1",
  passwordConfirmation: "Motdepasse1",
};

describe("validateNewPassword", () => {
  it("accepte un mot de passe conforme aux règles de l'API", () => {
    expect(validateNewPassword("Motdepasse1")).toBeUndefined();
  });

  it.each([
    ["vide", ""],
    ["trop court", "Abcdef1".padEnd(PASSWORD_MIN_LENGTH - 1, "a")],
    ["sans majuscule", "motdepasse1"],
    ["sans minuscule", "MOTDEPASSE1"],
    ["sans chiffre", "Motdepasses"],
  ])("refuse un mot de passe %s", (_, password) => {
    expect(validateNewPassword(password)).toBeDefined();
  });
});

describe("validateSignup", () => {
  it("ne signale aucune erreur pour une saisie valide", () => {
    expect(hasErrors(validateSignup(validValues))).toBe(false);
  });

  it("renvoie chaque champ validé, même valide, pour effacer une erreur précédente", () => {
    expect(validateSignup(validValues, ["email"])).toEqual({ email: undefined });
  });

  it.each([
    ["06.12.34.56.78", true],
    ["+33 6 12 34 56 78", true],
    ["01 23 45 67 89", false],
    ["+33", false],
  ])("numéro de portable %s valide : %s", (mobilePhone, valid) => {
    const { mobilePhone: error } = validateSignup({ ...validValues, mobilePhone }, ["mobilePhone"]);
    expect(error === undefined).toBe(valid);
  });

  it("applique le format de code postal du pays choisi", () => {
    const belgian = { ...validValues, country: "BE", postalCode: "1000" };
    expect(validateSignup(belgian, ["postalCode"]).postalCode).toBeUndefined();
    expect(validateSignup({ ...belgian, country: "FR" }, ["postalCode"]).postalCode).toBeDefined();
  });

  it("refuse une date de naissance dans le futur ou impossible", () => {
    expect(
      validateSignup({ ...validValues, birthDate: "2999-01-01" }, ["birthDate"]).birthDate,
    ).toBeDefined();
    expect(
      validateSignup({ ...validValues, birthDate: "1990-13-45" }, ["birthDate"]).birthDate,
    ).toBeDefined();
  });

  it("refuse une confirmation différente du mot de passe", () => {
    const errors = validateSignup({ ...validValues, passwordConfirmation: "Autre12345" });
    expect(errors.passwordConfirmation).toBeDefined();
  });
});

describe("firstInvalidField et stepOf", () => {
  it("renvoie le premier champ en erreur dans l'ordre du formulaire, et son étape", () => {
    const errors = validateSignup({ ...validValues, city: "", email: "pas-un-email" });
    expect(firstInvalidField(errors)).toBe("email");
    expect(stepOf("email")).toBe(1);
    expect(stepOf("city")).toBe(2);
  });
});

describe("readSignupValues et toSignupPayload", () => {
  it("retire les espaces sauf dans les mots de passe", () => {
    const formData = new FormData();
    formData.set("firstName", "  Camille ");
    formData.set("password", " Motdepasse1 ");
    const values = readSignupValues(formData);
    expect(values.firstName).toBe("Camille");
    expect(values.password).toBe(" Motdepasse1 ");
  });

  it("normalise l'email et le numéro, et regroupe l'adresse", () => {
    expect(toSignupPayload(validValues)).toMatchObject({
      email: "camille@exemple.fr",
      mobilePhone: "0612345678",
      address: { street: "12 rue de la Paix", postalCode: "75002", city: "Paris", country: "FR" },
    });
  });
});
