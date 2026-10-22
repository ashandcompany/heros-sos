import { describe, expect, it } from "vitest";
import type { ApiSchema } from "./http";
import { jsonResponse, mockFetch, withApi } from "./test-utils";

const incident: ApiSchema<"Incident"> = {
  id: "incident-1",
  typeId: 1,
  typeCode: "braquage",
  typeLabel: "Braquage",
  title: "Braquage à la banque",
  description: "Un individu masqué menace les clients.",
  status: "in_progress",
  priority: "high",
  moderation: "approved",
  citizensRescued: 0,
  location: { address: "1 place de la Bourse", city: "Lyon", latitude: 45.76, longitude: 4.83 },
  occurredAt: null,
  villainIds: ["villain-1"],
  villains: [{ id: "villain-1", alias: "Docteur Givre" }],
  resolvedAt: null,
  lead: null,
  createdAt: "2026-10-21T08:00:00Z",
};

const page = (items: ApiSchema<"Incident">[]) =>
  jsonResponse({ items, total: items.length, limit: 100, offset: 0 });

describe("getOngoingIncidents (API)", () => {
  it("ne demande que les incidents non résolus, sans session", async () => {
    const fetchMock = mockFetch(() => page([incident]));
    const { getOngoingIncidents } = await withApi(() => import("./incidents"));

    await getOngoingIncidents();

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toContain("&status=reported&status=in_progress&status=failed");
    expect(new Headers(init.headers).has("Cookie")).toBe(false);
  });

  it("convertit statut, priorité, lieu et vilain vers la forme du frontend", async () => {
    mockFetch(() => page([incident]));
    const { getOngoingIncidents } = await withApi(() => import("./incidents"));

    expect(await getOngoingIncidents()).toEqual([
      {
        id: "incident-1",
        title: "Braquage à la banque",
        type: { code: "braquage", label: "Braquage" },
        status: "en-cours",
        moderation: "valide",
        priority: "haute",
        location: {
          city: "Lyon",
          address: "1 place de la Bourse",
          coordinates: { lat: 45.76, lng: 4.83 },
        },
        reportedAt: "2026-10-21T08:00:00Z",
        resolvedAt: undefined,
        villain: { id: "villain-1", alias: "Docteur Givre" },
        hero: undefined,
        description: "Un individu masqué menace les clients.",
        occurredAt: undefined,
      },
    ]);
  });

  it("traite un incident en échec comme un nouveau signalement", async () => {
    mockFetch(() => page([{ ...incident, status: "failed" }]));
    const { getOngoingIncidents } = await withApi(() => import("./incidents"));

    expect((await getOngoingIncidents())[0].status).toBe("signale");
  });

  it("donne un titre aux incidents qui n'en ont pas, et ignore des coordonnées absentes", async () => {
    mockFetch(() =>
      page([
        {
          ...incident,
          title: "",
          villains: [],
          location: { ...incident.location, latitude: null, longitude: null },
        },
      ]),
    );
    const { getOngoingIncidents } = await withApi(() => import("./incidents"));

    const [converted] = await getOngoingIncidents();
    expect(converted.title).toBe("Braquage · Lyon");
    expect(converted.location.coordinates).toBeUndefined();
    expect(converted.villain).toBeNull();
  });

  it("filtre par ville sans tenir compte de la casse", async () => {
    mockFetch(() =>
      page([incident, { ...incident, id: "incident-2", location: { address: "", city: "Paris" } }]),
    );
    const { getOngoingIncidents } = await withApi(() => import("./incidents"));

    expect((await getOngoingIncidents("lyon")).map(({ id }) => id)).toEqual(["incident-1"]);
  });
});
