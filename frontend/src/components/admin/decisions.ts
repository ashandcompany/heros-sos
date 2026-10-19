import type { ModerationStatus } from "@/types";
import type { Decision } from "./ModerationForm";

/**
 * Décisions proposées selon l'état actuel : valider ou rejeter un élément en attente,
 * revenir sur une décision déjà prise sinon.
 */
export function decisionsFor(
  status: ModerationStatus,
  labels: { approve: string; reject: string; restore?: string },
): Decision[] {
  switch (status) {
    case "en-attente":
      return [
        { value: "valide", label: labels.approve, variant: "primary" },
        { value: "rejete", label: labels.reject, variant: "danger" },
      ];
    case "valide":
      return [{ value: "rejete", label: labels.reject, variant: "danger" }];
    case "rejete":
      return [{ value: "valide", label: labels.restore ?? labels.approve, variant: "secondary" }];
  }
}
