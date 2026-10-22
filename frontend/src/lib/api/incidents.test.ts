import { describe, expect, it } from "vitest";
import type { ApiSchema } from "./http";
import { errorResponse, jsonResponse, mockFetch, withApi } from "./test-utils";

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

describe("modération et rôle du déclarant (API)", () => {
  it("convertit la décision de modération et le rôle du déclarant", async () => {
    mockFetch(() => page([{ ...incident, moderation: "pending", reporterRole: "victim" }]));
    const { getOngoingIncidents } = await withApi(() => import("./incidents"));

    const [converted] = await getOngoingIncidents();
    expect(converted.moderation).toBe("en-attente");
    expect(converted.reporterRole).toBe("victime");
  });

  it("filtre la file de modération par décision", async () => {
    mockFetch(() =>
      page([
        { ...incident, id: "incident-1", moderation: "pending", interventionId: undefined },
        { ...incident, id: "incident-2", moderation: "rejected", interventionId: undefined },
      ]),
    );
    const { getAllIncidents } = await withApi(() => import("./incidents"));

    expect((await getAllIncidents("token", "rejete")).map(({ id }) => id)).toEqual(["incident-2"]);
  });
});

/** Corps JSON du premier appel à `path`. */
function sentBody(fetchMock: ReturnType<typeof mockFetch>, path: string) {
  const calls = fetchMock.mock.calls as unknown as [string, RequestInit][];
  const [, init] = calls.find(([url]) => url.endsWith(path)) ?? [];
  return { method: init?.method, body: JSON.parse(String(init?.body)) };
}

describe("mutations des incidents (API)", () => {
  it("modère un incident", async () => {
    const fetchMock = mockFetch(() => new Response(null, { status: 204 }));
    const { moderateIncident } = await withApi(() => import("./incidents"));

    expect(await moderateIncident("token", "incident-1", "rejete")).toEqual({ ok: true });
    expect(sentBody(fetchMock, "/admin/incidents/incident-1/moderation")).toEqual({
      method: "PUT",
      body: { decision: "rejected" },
    });
  });

  it("ne remet pas une décision en attente", async () => {
    const fetchMock = mockFetch(() => new Response(null, { status: 204 }));
    const { moderateIncident } = await withApi(() => import("./incidents"));

    expect((await moderateIncident("token", "incident-1", "en-attente")).ok).toBe(false);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("change la priorité, et explique le refus quand une intervention a commencé", async () => {
    let status = 204;
    const fetchMock = mockFetch(() =>
      status === 204
        ? new Response(null, { status })
        : errorResponse(status, "incident_not_available"),
    );
    const { setIncidentPriority } = await withApi(() => import("./incidents"));

    expect(await setIncidentPriority("token", "incident-1", "critique")).toEqual({ ok: true });
    expect(sentBody(fetchMock, "/hero/incidents/incident-1/priority")).toEqual({
      method: "PUT",
      body: { priority: "critical" },
    });

    status = 409;
    const refused = await setIncidentPriority("token", "incident-1", "haute");
    expect(refused).toEqual({ ok: false, error: expect.stringContaining("intervient déjà") });
  });

  it("corrige un incident en gardant la position quand le lieu ne change pas", async () => {
    // `GET /incidents/{id}` (position connue) puis la réponse du `PATCH`.
    const fetchMock = mockFetch(() => jsonResponse(incident));
    const { updateIncident } = await withApi(() => import("./incidents"));

    const result = await updateIncident("token", "incident-1", {
      title: "Braquage déjoué",
      typeId: 1,
      status: "resolu",
      description: "Les clients sont sains et saufs.",
      location: { address: "1 place de la Bourse", city: "lyon" },
      villainId: null,
      heroId: "hero-1",
    });

    expect(result).toEqual({ ok: true });
    expect(sentBody(fetchMock, "/admin/incidents/incident-1")).toEqual({
      method: "PATCH",
      body: {
        typeId: 1,
        title: "Braquage déjoué",
        description: "Les clients sont sains et saufs.",
        status: "succeeded",
        location: {
          address: "1 place de la Bourse",
          city: "lyon",
          latitude: 45.76,
          longitude: 4.83,
        },
        villainId: null,
        heroId: "hero-1",
      },
    });
  });
});
