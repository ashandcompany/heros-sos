"use client";

import Link from "next/link";
import { type ReactNode, useMemo, useRef, useState } from "react";
import { cn } from "@/components/ui/cn";
import { formatDateTime, incidentPriorityLabels, incidentStatusLabels } from "@/lib/format";
import type { MapTileConfig } from "@/lib/map";
import { heroIncidentRoute } from "@/lib/routes";
import type { IncidentMapData, IncidentPriority, MapIncident } from "@/types";
import { IncidentMap, type IncidentMapVariant } from "./IncidentMap";

const PRIORITIES = ["critique", "haute", "normale", "basse"] as const satisfies IncidentPriority[];

type StatusFilter = "tous" | MapIncident["status"];

const statusFilters: Array<{ value: StatusFilter; label: string }> = [
  { value: "tous", label: "Tous" },
  { value: "signale", label: "Signalés" },
  { value: "en-cours", label: "En cours" },
];

/** Hauteur commune de la carte et de la liste sur grand écran. */
const PANEL_HEIGHT = "lg:h-[min(calc(100vh-11rem),720px)]";

const plural = (count: number, word: string) => `${count} ${word}${count > 1 ? "s" : ""}`;

/** Case à cocher au style des filtres (zones, super-héros). */
function LayerToggle({
  checked,
  onChange,
  children,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  children: ReactNode;
}) {
  return (
    <label className="flex cursor-pointer items-center gap-2 text-sm">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="size-4 accent-primary"
      />
      {children}
    </label>
  );
}

/**
 * Carte des incidents et sa liste, synchronisées : choisir un incident dans la liste recadre la
 * carte sur son repère, cliquer un repère met l'incident en évidence dans la liste.
 * Les filtres (priorité, statut, couches) s'appliquent aux deux.
 */
export function IncidentMapExplorer({
  data,
  tiles,
  variant,
}: {
  data: IncidentMapData;
  tiles: MapTileConfig;
  variant: IncidentMapVariant;
}) {
  const [priorities, setPriorities] = useState<ReadonlySet<IncidentPriority>>(
    () => new Set(PRIORITIES),
  );
  const [status, setStatus] = useState<StatusFilter>("tous");
  const [showZones, setShowZones] = useState(true);
  const [showTeams, setShowTeams] = useState(true);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const mapRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLUListElement>(null);

  const teamByIncident = useMemo(
    () => new Map(data.teams.map((team) => [team.incidentId, team])),
    [data.teams],
  );

  const visible = useMemo(
    () =>
      data.incidents.filter(
        (incident) =>
          priorities.has(incident.priority) && (status === "tous" || incident.status === status),
      ),
    [data.incidents, priorities, status],
  );
  const visibleIds = useMemo(() => new Set(visible.map(({ id }) => id)), [visible]);

  // Un incident masqué par les filtres ne reste pas sélectionné.
  const activeId = selectedId && visibleIds.has(selectedId) ? selectedId : null;

  const countByPriority = useMemo(() => {
    const counts = new Map<IncidentPriority, number>();
    for (const { priority } of data.incidents)
      counts.set(priority, (counts.get(priority) ?? 0) + 1);
    return counts;
  }, [data.incidents]);

  const filtered = visible.length !== data.incidents.length;

  const togglePriority = (priority: IncidentPriority) =>
    setPriorities((current) => {
      const next = new Set(current);
      if (next.has(priority)) next.delete(priority);
      else next.add(priority);
      return next;
    });

  const resetFilters = () => {
    setPriorities(new Set(PRIORITIES));
    setStatus("tous");
  };

  /** Clic sur un repère : l'incident est mis en évidence et ramené dans la partie visible de la liste. */
  const selectFromMap = (id: string | null) => {
    setSelectedId(id);
    const list = listRef.current;
    const item = id && list?.querySelector<HTMLElement>(`[data-incident="${CSS.escape(id)}"]`);
    if (!list || !item || list.scrollHeight <= list.clientHeight) return;
    const top = item.offsetTop - list.offsetTop;
    if (top < list.scrollTop || top + item.offsetHeight > list.scrollTop + list.clientHeight) {
      list.scrollTo({ top: top - 8, behavior: "smooth" });
    }
  };

  /** Choix dans la liste : sur mobile, la carte est au-dessus, on la ramène à l'écran. */
  const selectFromList = (id: string) => {
    setSelectedId((current) => (current === id ? null : id));
    const map = mapRef.current;
    if (map && map.getBoundingClientRect().top < 0) {
      map.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(20rem,24rem)_1fr]">
      <div ref={mapRef} className="scroll-mt-32 lg:order-2">
        <IncidentMap
          data={data}
          tiles={tiles}
          variant={variant}
          visibleIds={visibleIds}
          showZones={showZones}
          showTeams={showTeams}
          selectedId={activeId}
          onSelect={selectFromMap}
          className={cn("h-[60vh] min-h-80", PANEL_HEIGHT)}
        />
        <p className="mt-3 text-xs text-muted">
          {variant === "public"
            ? "Positions approximatives (au quartier près) pour protéger les personnes concernées."
            : "La position d'un super-héros est celle de l'incident qu'il traite."}
          {data.unlocatedCount > 0 &&
            ` ${plural(data.unlocatedCount, "incident")} sans position connue n'${data.unlocatedCount > 1 ? "apparaissent" : "apparaît"} pas sur la carte.`}
        </p>
      </div>

      <section
        aria-labelledby="map-list-title"
        className={cn("flex flex-col border-2 border-edge bg-surface lg:order-1", PANEL_HEIGHT)}
      >
        <div className="space-y-3 border-b-2 border-edge p-4">
          <div className="flex items-baseline justify-between gap-3">
            <h2 id="map-list-title" className="font-display text-lg font-bold">
              Incidents
            </h2>
            <p role="status" className="font-mono text-xs text-muted">
              {filtered
                ? `${visible.length} sur ${data.incidents.length}`
                : variant === "public"
                  ? `${data.incidents.length} en cours`
                  : plural(data.incidents.length, "non résolu")}
            </p>
          </div>

          <fieldset>
            <legend className="mb-1.5 font-mono text-[0.6875rem] tracking-wider text-muted uppercase">
              Priorité
            </legend>
            <div className="flex flex-wrap gap-1.5">
              {PRIORITIES.map((priority) => {
                const active = priorities.has(priority);
                return (
                  <button
                    key={priority}
                    type="button"
                    aria-pressed={active}
                    onClick={() => togglePriority(priority)}
                    className={cn(
                      "flex items-center gap-1.5 border-2 px-2 py-1 font-mono text-xs font-semibold tracking-wide uppercase",
                      active
                        ? "border-edge bg-surface-raised text-foreground"
                        : "border-border text-muted line-through decoration-2 opacity-70 hover:opacity-100",
                    )}
                  >
                    <span
                      aria-hidden="true"
                      className={`incident-map__marker incident-map__marker--${priority} size-3`}
                    >
                      <span className="border-2!" />
                    </span>
                    {incidentPriorityLabels[priority]}
                    <span className="text-muted">{countByPriority.get(priority) ?? 0}</span>
                  </button>
                );
              })}
            </div>
          </fieldset>

          <fieldset>
            <legend className="mb-1.5 font-mono text-[0.6875rem] tracking-wider text-muted uppercase">
              Statut
            </legend>
            <div className="grid grid-cols-3 border-2 border-edge">
              {statusFilters.map(({ value, label }) => (
                <button
                  key={value}
                  type="button"
                  aria-pressed={status === value}
                  onClick={() => setStatus(value)}
                  className={cn(
                    "px-2 py-1.5 font-mono text-xs font-semibold tracking-wide uppercase not-first:border-l-2 not-first:border-edge",
                    status === value
                      ? "bg-primary text-primary-foreground"
                      : "text-muted hover:bg-surface-raised hover:text-foreground",
                  )}
                >
                  {label}
                </button>
              ))}
            </div>
          </fieldset>

          <div className="flex flex-wrap gap-x-5 gap-y-1">
            <LayerToggle checked={showZones} onChange={setShowZones}>
              Zones touchées
            </LayerToggle>
            {variant === "hero" && (
              <LayerToggle checked={showTeams} onChange={setShowTeams}>
                Super-héros
              </LayerToggle>
            )}
          </div>
        </div>

        {visible.length === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center text-sm text-muted">
            {data.incidents.length === 0 ? (
              <p>Aucun incident en cours pour le moment.</p>
            ) : (
              <>
                <p>Aucun incident ne correspond à ces filtres.</p>
                <button
                  type="button"
                  onClick={resetFilters}
                  className="font-semibold text-foreground underline underline-offset-2 hover:text-accent"
                >
                  Réinitialiser les filtres
                </button>
              </>
            )}
          </div>
        ) : (
          <ul ref={listRef} className="relative flex-1 divide-y divide-border overflow-y-auto">
            {visible.map((incident) => {
              const selected = incident.id === activeId;
              const team = teamByIncident.get(incident.id);
              return (
                <li key={incident.id} data-incident={incident.id} className="relative">
                  <button
                    type="button"
                    aria-pressed={selected}
                    onClick={() => selectFromList(incident.id)}
                    className={cn(
                      "flex w-full gap-3 border-l-4 py-3 pr-4 pl-3 text-left",
                      selected
                        ? "border-l-primary bg-surface-raised"
                        : "border-l-transparent hover:bg-surface-raised/60",
                    )}
                  >
                    <span
                      aria-hidden="true"
                      className={`incident-map__marker incident-map__marker--${incident.priority} incident-map__marker--${incident.status} mt-1 size-3.5 shrink-0`}
                    >
                      <span className="border-2!" />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block leading-snug font-semibold">{incident.title}</span>
                      <span className="mt-0.5 block truncate text-sm text-muted">
                        {incident.type.label} · {incident.place}
                      </span>
                      <span className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-[0.6875rem] tracking-wide text-muted uppercase">
                        <span className="sr-only">
                          Priorité {incidentPriorityLabels[incident.priority]},{" "}
                        </span>
                        <span
                          className={cn(
                            "font-semibold",
                            incident.status === "en-cours" ? "text-rating" : "text-foreground",
                          )}
                        >
                          {incidentStatusLabels[incident.status]}
                        </span>
                        <time dateTime={incident.reportedAt} className="whitespace-nowrap">
                          {formatDateTime(incident.reportedAt)}
                        </time>
                      </span>
                      {team && (
                        <span
                          className={cn(
                            "mt-1 block text-xs",
                            team.isMe ? "font-semibold text-accent" : "text-muted",
                          )}
                        >
                          {team.isMe
                            ? "Vous intervenez"
                            : `L'équipe de ${team.hero.alias} intervient`}
                        </span>
                      )}
                    </span>
                  </button>
                  {variant === "hero" && (
                    <Link
                      href={heroIncidentRoute(incident.id)}
                      className="absolute right-3 bottom-3 font-mono text-[0.6875rem] font-semibold tracking-wide text-accent uppercase hover:underline"
                    >
                      Dossier →
                    </Link>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </div>
  );
}
