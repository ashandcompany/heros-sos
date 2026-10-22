import { describe, expect, it } from "vitest";
import type { ApiSchema } from "./http";
import { errorResponse, jsonResponse, mockFetch, withApi } from "./test-utils";

describe("getVillain (API)", () => {
  it("complète les champs facultatifs absents", async () => {
    const villain: ApiSchema<"Villain"> = {
      id: "villain-1",
      alias: "Docteur Givre",
      typeIds: [2],
      specialties: [{ id: 2, code: "gel", label: "Gel" }],
      threatLevel: "eleve",
      status: "en-liberte",
      reportedIncidentsCount: 4,
      createdAt: "2026-01-01T00:00:00Z",
    };
    mockFetch(() => jsonResponse(villain));
    const { getVillain } = await withApi(() => import("./villains"));

    expect(await getVillain("villain-1")).toEqual({
      id: "villain-1",
      alias: "Docteur Givre",
      threatLevel: "eleve",
      description: "",
      reportedIncidentsCount: 4,
      status: "en-liberte",
      lastName: "",
      firstName: "",
      mainSector: "",
      typeIds: [2],
      specialties: [{ id: 2, code: "gel", label: "Gel" }],
    });
  });

  it("renvoie null pour un vilain inconnu (404)", async () => {
    mockFetch(() => errorResponse(404, "villain_not_found"));
    const { getVillain } = await withApi(() => import("./villains"));

    expect(await getVillain("inconnu")).toBeNull();
  });

  it("laisse remonter une autre erreur de l'API", async () => {
    mockFetch(() => errorResponse(500, "internal"));
    const { getVillain } = await withApi(() => import("./villains"));

    await expect(getVillain("villain-1")).rejects.toMatchObject({ status: 500 });
  });
});

describe("getSuperHero (API)", () => {
  it("convertit la fiche publique d'un héros", async () => {
    const hero: ApiSchema<"PublicHero"> = {
      id: "hero-1",
      alias: "Lumen",
      avatarUrl: null,
      power: "Contrôle de la lumière",
      averageRating: 4.5,
      interventionsCount: 12,
    };
    mockFetch(() => jsonResponse(hero));
    const { getSuperHero } = await withApi(() => import("./super-heroes"));

    expect(await getSuperHero("hero-1")).toEqual(hero);
  });
});
