import { statusTone } from "@/components/citizen/IncidentListItem";
import { Badge } from "@/components/ui/Badge";
import { incidentPriorityLabels, incidentStatusLabels } from "@/lib/format";
import type { IncidentPriority, IncidentStatus } from "@/types";

const priorityTone = {
  basse: "sage",
  normale: "neutral",
  haute: "warning",
  critique: "danger",
} as const satisfies Record<IncidentPriority, string>;

export function PriorityBadge({ priority }: { priority: IncidentPriority }) {
  return (
    <Badge tone={priorityTone[priority]}>
      <span className="sr-only">Priorité </span>
      {incidentPriorityLabels[priority]}
    </Badge>
  );
}

export function StatusBadge({ status }: { status: IncidentStatus }) {
  return <Badge tone={statusTone[status]}>{incidentStatusLabels[status]}</Badge>;
}
