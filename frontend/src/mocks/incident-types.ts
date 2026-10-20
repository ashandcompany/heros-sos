import type { IncidentType, IncidentTypeDefinition } from "@/types";

/** Référentiel des types d'incident (mock en mémoire, complété par la modération). */
export const incidentTypesMock: IncidentTypeDefinition[] = [
  { id: 1, code: "vol", label: "Vol", endangersLife: false },
  { id: 2, code: "cambriolage", label: "Cambriolage", endangersLife: false },
  { id: 3, code: "prise-d-otages", label: "Prise d'otages", endangersLife: true },
  { id: 4, code: "agression", label: "Agression", endangersLife: true },
  { id: 5, code: "sabotage", label: "Sabotage", endangersLife: false },
  { id: 6, code: "enlevement", label: "Enlèvement", endangersLife: true },
];

/** Type embarqué dans un incident mock, à partir de son code. */
export function mockIncidentType(code: string): IncidentType {
  const type = incidentTypesMock.find((existing) => existing.code === code);
  if (!type) throw new Error(`Type d'incident mock inconnu : ${code}`);
  return { code: type.code, label: type.label };
}
