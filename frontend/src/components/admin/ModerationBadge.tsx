import { Badge } from "@/components/ui/Badge";
import { moderationStatusLabels } from "@/lib/format";
import type { ModerationStatus } from "@/types";

const tones = { "en-attente": "warning", valide: "success", rejete: "danger" } as const;

/** Décision de modération ; `labels` adapte le vocabulaire (ex : « Suspendu » pour un compte). */
export function ModerationBadge({
  status,
  labels,
}: {
  status: ModerationStatus;
  labels?: Partial<Record<ModerationStatus, string>>;
}) {
  return <Badge tone={tones[status]}>{labels?.[status] ?? moderationStatusLabels[status]}</Badge>;
}
