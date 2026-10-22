import { cn } from "@/components/ui/cn";

/**
 * Légende de `IncidentMap`, posée sur la carte : mêmes classes que les repères dessinés par
 * Leaflet. Repliable pour libérer la carte sur petit écran.
 */
export function MapLegend({
  showTeams = false,
  className,
}: {
  showTeams?: boolean;
  className?: string;
}) {
  return (
    <details
      open
      className={cn(
        "group border-2 border-ink bg-paper text-ink shadow-[3px_3px_0_0_var(--color-ink)]",
        className,
      )}
    >
      <summary className="flex cursor-pointer list-none items-center justify-between gap-3 px-2.5 py-1.5 font-mono text-[0.6875rem] font-semibold tracking-wider uppercase [&::-webkit-details-marker]:hidden">
        Légende
        <span aria-hidden="true" className="transition-transform group-open:rotate-180">
          ▾
        </span>
      </summary>
      <ul className="space-y-1.5 border-t-2 border-dashed border-ink/30 px-2.5 py-2 text-xs">
        <li className="flex items-center gap-2">
          <span aria-hidden="true" className="flex gap-1">
            {(["basse", "normale", "haute", "critique"] as const).map((priority) => (
              <span
                key={priority}
                className={`incident-map__marker incident-map__marker--${priority} size-3.5`}
              >
                <span />
              </span>
            ))}
          </span>
          Priorité, de basse à critique
        </li>
        <li className="flex items-center gap-2">
          <span
            aria-hidden="true"
            className="incident-map__marker incident-map__marker--haute incident-map__marker--en-cours size-3.5"
          >
            <span />
          </span>
          Intervention en cours
        </li>
        {showTeams && (
          <li className="flex items-center gap-2">
            <span aria-hidden="true" className="incident-map__team size-5 text-[0.6rem]">
              H
            </span>
            Super-héros en intervention
          </li>
        )}
        <li className="flex items-center gap-2">
          <span
            aria-hidden="true"
            className="size-3.5 border-2 border-dashed border-brick bg-brick/20"
          />
          Zone touchée (par ville)
        </li>
      </ul>
    </details>
  );
}
