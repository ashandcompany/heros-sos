import type { HeroIntervention } from "@/types";

/**
 * Historique des interventions (mock en mémoire). Une intervention sans `outcome` est en cours ;
 * l'incident correspondant est alors « en-cours » avec ce héros.
 */
export const interventionsMock: HeroIntervention[] = [
  {
    id: "intervention-1",
    incidentId: "incident-14",
    hero: { id: "hero-1", alias: "Lumen" },
    startedAt: "2026-10-02T02:25:00+02:00",
    endedAt: "2026-10-02T04:45:00+02:00",
    outcome: "reussi",
  },
  {
    id: "intervention-2",
    incidentId: "incident-13",
    hero: { id: "hero-1", alias: "Lumen" },
    startedAt: "2026-10-09T18:41:00+02:00",
    endedAt: "2026-10-09T19:02:00+02:00",
    outcome: "reussi",
  },
  {
    id: "intervention-3",
    incidentId: "incident-1",
    hero: { id: "hero-1", alias: "Lumen" },
    startedAt: "2026-10-17T14:20:00+02:00",
    endedAt: "2026-10-17T14:41:00+02:00",
    outcome: "reussi",
  },
  {
    id: "intervention-4",
    incidentId: "incident-8",
    hero: { id: "hero-1", alias: "Lumen" },
    startedAt: "2026-10-18T21:10:00+02:00",
    endedAt: "2026-10-18T21:55:00+02:00",
    outcome: "echoue",
  },
  {
    id: "intervention-5",
    incidentId: "incident-8",
    hero: { id: "hero-7", alias: "Brume" },
    startedAt: "2026-10-18T22:20:00+02:00",
    endedAt: null,
    outcome: null,
  },
  {
    id: "intervention-6",
    incidentId: "incident-10",
    hero: { id: "hero-3", alias: "Vélocité" },
    startedAt: "2026-10-19T07:44:00+02:00",
    endedAt: null,
    outcome: null,
  },
  {
    id: "intervention-7",
    incidentId: "incident-12",
    hero: { id: "hero-1", alias: "Lumen" },
    startedAt: "2026-10-19T10:12:00+02:00",
    endedAt: null,
    outcome: null,
  },
];
