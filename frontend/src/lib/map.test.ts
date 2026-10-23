import { describe, expect, it, vi } from "vitest";
import { getMapTileConfig } from "./map";

describe("getMapTileConfig", () => {
  it("utilise les tuiles d'OpenStreetMap sans configuration", () => {
    vi.stubEnv("MAP_TILE_URL", "");
    expect(getMapTileConfig().url).toBe("https://tile.openstreetmap.org/{z}/{x}/{y}.png");
  });

  it("lit le fournisseur configuré au démarrage", () => {
    vi.stubEnv("MAP_TILE_URL", " https://tuiles.exemple.fr/{z}/{x}/{y}.png ");
    vi.stubEnv("MAP_TILE_ATTRIBUTION", "© Exemple");
    vi.stubEnv("MAP_TILE_MAX_ZOOM", "17");
    expect(getMapTileConfig()).toEqual({
      url: "https://tuiles.exemple.fr/{z}/{x}/{y}.png",
      attribution: "© Exemple",
      maxZoom: 17,
    });
  });

  it.each(["", "0", "18.5", "abc"])("retombe sur le zoom 19 pour MAP_TILE_MAX_ZOOM=%j", (zoom) => {
    vi.stubEnv("MAP_TILE_URL", "https://tuiles.exemple.fr/{z}/{x}/{y}.png");
    vi.stubEnv("MAP_TILE_MAX_ZOOM", zoom);
    expect(getMapTileConfig().maxZoom).toBe(19);
  });
});
