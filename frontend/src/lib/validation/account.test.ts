import { describe, expect, it } from "vitest";
import {
  AVATAR_MAX_BYTES,
  emailChangeFailureState,
  validateAvatar,
  validateContact,
  validateEmail,
} from "./account";

const image = (size: number, type: string) => new File([new Uint8Array(size)], "photo", { type });

describe("validateAvatar", () => {
  it("accepte une image JPEG, PNG ou WebP de 2 Mo au plus", () => {
    expect(validateAvatar(image(AVATAR_MAX_BYTES, "image/webp"))).toBeUndefined();
  });

  it.each([
    ["absente", null],
    ["vide", image(0, "image/png")],
    ["d'un autre format", image(10, "image/gif")],
    ["trop lourde", image(AVATAR_MAX_BYTES + 1, "image/jpeg")],
  ])("refuse une image %s", (_, file) => {
    expect(validateAvatar(file)).toBeDefined();
  });
});

describe("validateContact (coordonnées)", () => {
  const values = {
    mobilePhone: "0612345678",
    street: "12 rue de la Paix",
    complement: "",
    postalCode: "75002",
    city: "Paris",
    country: "FR",
  };

  it("reprend les règles de l'inscription", () => {
    expect(Object.values(validateContact(values)).some(Boolean)).toBe(false);
    expect(validateContact({ ...values, postalCode: "7500" }).postalCode).toBeDefined();
  });

  it("borne le complément d'adresse", () => {
    expect(validateContact({ ...values, complement: "a".repeat(101) }).complement).toBeDefined();
  });
});

describe("changement d'email", () => {
  it("valide la nouvelle adresse", () => {
    expect(validateEmail("")).toBeDefined();
    expect(validateEmail("camille@exemple")).toBeDefined();
    expect(validateEmail("camille@exemple.fr")).toBeUndefined();
  });

  it("place les refus de l'API sur le champ email", () => {
    const state = emailChangeFailureState(
      { ok: false, error: "email-already-used" },
      "pris@exemple.fr",
    );
    expect(state.fieldErrors.email).toBeDefined();
    expect(state.values).toEqual({ email: "pris@exemple.fr" });
  });
});
