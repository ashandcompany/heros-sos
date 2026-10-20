/** Légende de `IncidentMap` : mêmes classes que les repères dessinés par Leaflet. */
export function MapLegend({ showTeams = false }: { showTeams?: boolean }) {
  return (
    <ul aria-label="Légende" className="flex flex-wrap gap-x-6 gap-y-2 text-sm">
      <li className="flex items-center gap-2">
        <span aria-hidden="true" className="flex gap-1">
          {(["basse", "normale", "haute", "critique"] as const).map((priority) => (
            <span
              key={priority}
              className={`incident-map__marker incident-map__marker--${priority} size-4`}
            >
              <span />
            </span>
          ))}
        </span>
        Incident signalé, par priorité (basse à critique)
      </li>
      <li className="flex items-center gap-2">
        <span
          aria-hidden="true"
          className="incident-map__marker incident-map__marker--haute incident-map__marker--en-cours size-4"
        >
          <span />
        </span>
        Incident pris en charge
      </li>
      {showTeams && (
        <li className="flex items-center gap-2">
          <span aria-hidden="true" className="incident-map__team size-6">
            H
          </span>
          Super-héros en intervention
        </li>
      )}
      <li className="flex items-center gap-2">
        <span
          aria-hidden="true"
          className="size-5 border-2 border-dashed border-brick bg-brick/20"
        />
        Zone touchée (incidents non résolus d&apos;une ville)
      </li>
    </ul>
  );
}
