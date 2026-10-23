import { describe, expect, it } from "vitest";
import {
  ARTICLE_CONTENT_MIN_LENGTH,
  ARTICLE_TITLE_MAX_LENGTH,
  INCIDENT_TYPE_CODE_MAX_LENGTH,
  parseVillainTypeIds,
  toIncidentUpdate,
  toVillainInput,
  validateArticle,
  validateIncidentEdit,
  validateIncidentType,
  validateVillain,
  visibleTextLength,
  type IncidentEditValues,
  type VillainValues,
} from "./admin";

describe("visibleTextLength", () => {
  it("ne compte que le texte visible (ni balises, ni entités, ni espaces répétés)", () => {
    expect(visibleTextLength("<p>Bonjour&nbsp;<strong>le</strong>   monde</p>")).toBe(
      "Bonjour le monde".length,
    );
    expect(visibleTextLength("<p></p><br>")).toBe(0);
  });
});

describe("validateArticle", () => {
  const article = {
    title: "Se protéger d'un rayon gelant",
    excerpt: "Les bons réflexes.",
    content: "<p>Restez à couvert et prévenez Heros-SOS.</p>",
    status: "publie",
  };

  it("accepte un article valide", () => {
    expect(Object.values(validateArticle(article)).some(Boolean)).toBe(false);
  });

  it("mesure le contenu sur le texte visible, pas sur le HTML", () => {
    const content = `<p><em>${"a".repeat(ARTICLE_CONTENT_MIN_LENGTH - 1)}</em></p>`;
    expect(validateArticle({ ...article, content }).content).toBeDefined();
  });

  it("refuse un titre trop long, un chapeau vide et un statut inconnu", () => {
    const errors = validateArticle({
      ...article,
      title: "a".repeat(ARTICLE_TITLE_MAX_LENGTH + 1),
      excerpt: "",
      status: "archive",
    });
    expect(errors.title).toBeDefined();
    expect(errors.excerpt).toBeDefined();
    expect(errors.status).toBeDefined();
  });
});

describe("validateIncidentType", () => {
  it.each([
    ["prise-d-otages", true],
    ["vol2", true],
    ["Prise", false],
    ["prise--otages", false],
    ["-prise", false],
    ["évasion", false],
    ["a".repeat(INCIDENT_TYPE_CODE_MAX_LENGTH + 1), false],
  ])("code %s valide : %s", (code, valid) => {
    const { code: error } = validateIncidentType({ code, label: "Libellé", endangersLife: "" });
    expect(error === undefined).toBe(valid);
  });
});

describe("validateIncidentEdit et toIncidentUpdate", () => {
  const values: IncidentEditValues = {
    title: "Braquage en cours",
    type: "1",
    status: "en-cours",
    description: "",
    address: "",
    city: "Lyon",
    villainId: "",
    heroId: "",
  };
  const options = { villainIds: [], heroIds: ["hero-1"], typeIds: [1] };

  it("exige un héros dès que l'incident n'est plus « signalé »", () => {
    expect(validateIncidentEdit(values, options).heroId).toBeDefined();
    expect(validateIncidentEdit({ ...values, heroId: "hero-1" }, options).heroId).toBeUndefined();
    expect(validateIncidentEdit({ ...values, status: "signale" }, options).heroId).toBeUndefined();
  });

  it("retire le héros d'un incident repassé à « signalé »", () => {
    expect(toIncidentUpdate({ ...values, status: "signale", heroId: "hero-1" }).heroId).toBeNull();
  });
});

describe("villains", () => {
  const values: VillainValues = {
    alias: "Docteur Givre",
    firstName: "",
    lastName: "",
    mainSector: "",
    description: "",
    typeIds: "2,1,2",
    threatLevel: "eleve",
    status: "en-liberte",
  };

  it("dédoublonne les spécialités cochées", () => {
    expect(parseVillainTypeIds("2,1,2")).toEqual([2, 1]);
    expect(parseVillainTypeIds("")).toEqual([]);
    expect(toVillainInput(values).typeIds).toEqual([2, 1]);
  });

  it("refuse une spécialité supprimée entre-temps", () => {
    expect(validateVillain(values, { typeIds: [1, 2] }).typeIds).toBeUndefined();
    expect(validateVillain(values, { typeIds: [1] }).typeIds).toBeDefined();
  });

  it("refuse un niveau de menace ou un statut inconnu", () => {
    const errors = validateVillain(
      { ...values, threatLevel: "extreme", status: "mort" },
      { typeIds: [1, 2] },
    );
    expect(errors.threatLevel).toBeDefined();
    expect(errors.status).toBeDefined();
  });
});
