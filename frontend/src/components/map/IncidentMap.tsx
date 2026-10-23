"use client";

import "leaflet/dist/leaflet.css";
import type {
  LatLngBounds,
  LatLngTuple,
  LayerGroup,
  Map as LeafletMap,
  Marker,
  Popup,
} from "leaflet";
import { useEffect, useRef, useState } from "react";
import { cn } from "@/components/ui/cn";
import { formatDateTime, incidentPriorityLabels, incidentStatusLabels } from "@/lib/format";
import type { MapTileConfig } from "@/lib/map";
import { heroIncidentRoute } from "@/lib/routes";
import type { IncidentMapData, MapIncident, MapTeam, MapZone } from "@/types";
import { MapLegend } from "./MapLegend";

/** Vue de la France métropolitaine, quand aucun incident n'est localisé. */
const FRANCE_CENTER: LatLngTuple = [46.6, 2.4];
const FRANCE_ZOOM = 6;

/**
 * Cadrage initial : les incidents à plus de cette distance (en mètres) du point médian sont
 * laissés hors champ, pour qu'un incident isolé (à New York…) n'écrase pas tous les autres.
 * Ils restent sur la carte et dans la liste.
 */
const MAIN_AREA_RADIUS = 1_200_000;

/** Zoom minimal quand on sélectionne un incident depuis la liste. */
const FOCUS_ZOOM = 11;

/** Bulles : assez larges pour un titre sur deux lignes, sans masquer toute la carte. */
const POPUP_OPTIONS = {
  minWidth: 240,
  maxWidth: 300,
  autoPanPadding: [24, 24] as [number, number],
};

interface PopupCard {
  /** Petite ligne au-dessus du titre (type d'incident…). */
  eyebrow: string;
  title: string;
  badges?: Array<{ label: string; tone: string }>;
  facts: Array<{ label: string; value: string | null }>;
  link?: { href: string; label: string };
}

/** Crée un élément avec sa classe et son texte (jamais de HTML : les titres viennent des citoyens). */
function element<K extends keyof HTMLElementTagNameMap>(tag: K, className: string, text?: string) {
  const node = document.createElement(tag);
  node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

/** Fiche affichée dans une bulle, sur le modèle des dossiers du site (papier, liseré orange). */
function popupCard({ eyebrow, title, badges = [], facts, link }: PopupCard) {
  const card = element("article", "map-card");
  card.append(element("p", "map-card__eyebrow", eyebrow), element("h3", "map-card__title", title));

  if (badges.length > 0) {
    const row = element("p", "map-card__badges");
    for (const { label, tone } of badges) {
      row.append(element("span", `map-card__badge map-card__badge--${tone}`, label));
    }
    card.append(row);
  }

  const list = element("dl", "map-card__facts");
  for (const { label, value } of facts) {
    if (!value) continue;
    const fact = element("div", "map-card__fact");
    fact.append(element("dt", "", label), element("dd", "", value));
    list.append(fact);
  }
  card.append(list);

  if (link) {
    const anchor = element("a", "map-card__link", link.label);
    anchor.href = link.href;
    card.append(anchor);
  }
  return card;
}

/**
 * `hero` : carte opérationnelle de l'espace pro (liens vers les incidents, héros en intervention).
 * `public` : carte du site public, sans lien ni héros, avec des positions approximatives.
 */
export type IncidentMapVariant = "hero" | "public";

/** Carte publique : les positions sont arrondies, inutile de zoomer plus près que le quartier. */
const PUBLIC_MAX_ZOOM = 14;

const incidentPopup = (incident: MapIncident, variant: IncidentMapVariant) =>
  popupCard({
    eyebrow: incident.type.label,
    title: incident.title,
    badges: [
      { label: incidentPriorityLabels[incident.priority], tone: incident.priority },
      { label: incidentStatusLabels[incident.status], tone: incident.status },
    ],
    facts: [
      {
        label: "Lieu",
        value: variant === "public" ? `${incident.place} · approximatif` : incident.place,
      },
      { label: "Signalé", value: formatDateTime(incident.reportedAt) },
    ],
    link:
      variant === "hero"
        ? { href: heroIncidentRoute(incident.id), label: "Ouvrir le dossier" }
        : undefined,
  });

const teamPopup = (team: MapTeam) =>
  popupCard({
    eyebrow: "Super-héros",
    title: team.hero.alias,
    badges: [{ label: "En intervention", tone: "en-cours" }],
    facts: [{ label: "Depuis", value: team.startedAt && formatDateTime(team.startedAt) }],
    link: { href: heroIncidentRoute(team.incidentId), label: "Voir l'intervention" },
  });

const zonePopup = (zone: MapZone) =>
  popupCard({
    eyebrow: "Zone touchée",
    title: zone.city,
    facts: [
      {
        label: "Non résolus",
        value: `${zone.incidentsCount} incident${zone.incidentsCount > 1 ? "s" : ""}`,
      },
    ],
  });

const median = (values: number[]) => {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.floor(sorted.length / 2)];
};

interface MapLayers {
  map: LeafletMap;
  incidents: LayerGroup;
  zones: LayerGroup;
  teams: LayerGroup;
  markers: Map<string, Marker>;
  popups: Map<string, Popup>;
  /** Cadrage de la vue d'ensemble (`null` : aucun incident localisé). */
  home: LatLngBounds | null;
  animate: boolean;
}

export interface IncidentMapProps {
  data: IncidentMapData;
  tiles: MapTileConfig;
  variant?: IncidentMapVariant;
  /** Incidents affichés (filtres de la liste). */
  visibleIds: ReadonlySet<string>;
  showZones: boolean;
  showTeams: boolean;
  selectedId: string | null;
  /** Appelé au clic sur un repère (`id`) ou à la fermeture de sa bulle (`null`). */
  onSelect: (id: string | null) => void;
  className?: string;
}

/**
 * Carte Leaflet des incidents non résolus, des zones touchées et, dans l'espace pro, des
 * super-héros en intervention.
 * Leaflet manipule `window` dès son chargement : il est importé dans l'effet, côté navigateur.
 * La carte n'est pas accessible à tous : la page affiche aussi la liste des incidents.
 */
export function IncidentMap({
  data,
  tiles,
  variant = "hero",
  visibleIds,
  showZones,
  showTeams,
  selectedId,
  onSelect,
  className,
}: IncidentMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const layersRef = useRef<MapLayers | null>(null);
  const [ready, setReady] = useState(false);

  // Rappel le plus récent, sans recréer la carte à chaque rendu du parent.
  const onSelectRef = useRef(onSelect);
  useEffect(() => {
    onSelectRef.current = onSelect;
  });
  // Incident sélectionné, lu par les écouteurs Leaflet (créés une seule fois).
  const selectedRef = useRef<string | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let cancelled = false;

    void import("leaflet").then(({ default: L }) => {
      if (cancelled) return;

      const animate = !window.matchMedia("(prefers-reduced-motion: reduce)").matches;
      const map = L.map(container, {
        center: FRANCE_CENTER,
        zoom: FRANCE_ZOOM,
        // La molette ferait zoomer la carte en faisant défiler la page.
        scrollWheelZoom: false,
        zoomAnimation: animate,
        fadeAnimation: animate,
        markerZoomAnimation: animate,
        maxZoom: variant === "public" ? Math.min(PUBLIC_MAX_ZOOM, tiles.maxZoom) : tiles.maxZoom,
      });
      map.attributionControl.setPrefix(false);
      map.zoomControl.setPosition("topright");

      L.tileLayer(tiles.url, {
        attribution: tiles.attribution,
        maxZoom: tiles.maxZoom,
      }).addTo(map);

      const zones = L.layerGroup(
        data.zones.map((zone) =>
          L.circle([zone.center.lat, zone.center.lng], {
            radius: zone.radius,
            className: "incident-map__zone",
            interactive: true,
          }).bindPopup(() => zonePopup(zone), POPUP_OPTIONS),
        ),
      );

      // Les bulles des incidents ne s'ouvrent pas au clic (comportement de Leaflet) : le clic
      // sélectionne l'incident, et la sélection recadre la carte puis ouvre la bulle.
      const markers = new Map<string, Marker>();
      const popups = new Map<string, Popup>();
      const incidentByPopup = new Map<Popup, string>();
      for (const incident of data.incidents) {
        const marker = L.marker([incident.position.lat, incident.position.lng], {
          title: `${incident.title} (${incidentStatusLabels[incident.status]})`,
          riseOnHover: true,
          icon: L.divIcon({
            className: `incident-map__marker incident-map__marker--${incident.priority} incident-map__marker--${incident.status}`,
            html: "<span></span>",
            iconSize: [22, 22],
            iconAnchor: [11, 11],
            popupAnchor: [0, -12],
          }),
        });
        // Décalage d'une bulle attachée au repère : ancre de l'icône (-12) + décalage par défaut (7).
        const popup = L.popup({ ...POPUP_OPTIONS, offset: [0, -5] })
          .setLatLng(marker.getLatLng())
          .setContent(() => incidentPopup(incident, variant));

        marker.on("click", () => onSelectRef.current(incident.id));
        markers.set(incident.id, marker);
        popups.set(incident.id, popup);
        incidentByPopup.set(popup, incident.id);
      }

      // Bulle fermée (croix, clic sur la carte) : l'incident n'est plus sélectionné. Une bulle
      // fermée parce qu'un autre incident vient d'être choisi ne compte pas.
      map.on("popupclose", ({ popup }) => {
        const id = incidentByPopup.get(popup);
        if (id && id === selectedRef.current) onSelectRef.current(null);
      });

      const teams = L.layerGroup(
        data.teams.map((team) =>
          L.marker([team.position.lat, team.position.lng], {
            title: `${team.hero.alias}, en intervention`,
            riseOnHover: true,
            zIndexOffset: 500,
            icon: L.divIcon({
              className: "incident-map__team",
              html: '<span aria-hidden="true">H</span>',
              iconSize: [26, 26],
              // Décalé en haut à droite de l'incident qu'il traite, pour ne pas le masquer.
              iconAnchor: [-4, 30],
              popupAnchor: [17, -30],
            }),
          }).bindPopup(() => teamPopup(team), POPUP_OPTIONS),
        ),
      );

      const incidents = L.layerGroup().addTo(map);

      // Vue d'ensemble : la zone où se concentrent les incidents, sans les points isolés.
      let home: LatLngBounds | null = null;
      if (data.incidents.length > 0) {
        const center = L.latLng(
          median(data.incidents.map(({ position }) => position.lat)),
          median(data.incidents.map(({ position }) => position.lng)),
        );
        const points: LatLngTuple[] = data.incidents.map(({ position }) => [
          position.lat,
          position.lng,
        ]);
        const main = points.filter((point) => center.distanceTo(point) <= MAIN_AREA_RADIUS);
        home = L.latLngBounds(main.length > 0 ? main : points);
        map.fitBounds(home, { padding: [40, 40], maxZoom: variant === "public" ? 12 : 14 });
      }

      layersRef.current = { map, incidents, zones, teams, markers, popups, home, animate };
      setReady(true);
    });

    return () => {
      cancelled = true;
      layersRef.current?.map.remove();
      layersRef.current = null;
      setReady(false);
    };
  }, [data, tiles, variant]);

  // Filtres : seuls les repères des incidents retenus restent sur la carte.
  useEffect(() => {
    const layers = layersRef.current;
    if (!ready || !layers) return;
    for (const [id, marker] of layers.markers) {
      if (visibleIds.has(id)) layers.incidents.addLayer(marker);
      else layers.incidents.removeLayer(marker);
    }
  }, [ready, visibleIds]);

  useEffect(() => {
    const layers = layersRef.current;
    if (!ready || !layers) return;
    if (showZones) layers.map.addLayer(layers.zones);
    else layers.map.removeLayer(layers.zones);
  }, [ready, showZones]);

  useEffect(() => {
    const layers = layersRef.current;
    if (!ready || !layers) return;
    if (showTeams && variant === "hero") layers.map.addLayer(layers.teams);
    else layers.map.removeLayer(layers.teams);
  }, [ready, showTeams, variant]);

  // Sélection (liste ou repère) : on zoome sur le repère, puis on ouvre sa bulle.
  useEffect(() => {
    const layers = layersRef.current;
    selectedRef.current = selectedId;
    if (!ready || !layers) return;

    const { map, markers, popups, animate } = layers;
    for (const [id, marker] of markers) {
      marker.getElement()?.classList.toggle("is-selected", id === selectedId);
    }

    map.closePopup();
    const marker = selectedId ? markers.get(selectedId) : undefined;
    const popup = selectedId ? popups.get(selectedId) : undefined;
    if (!marker || !popup) return;

    const target = marker.getLatLng();
    const zoom = Math.max(map.getZoom(), Math.min(FOCUS_ZOOM, map.getMaxZoom()));
    const open = () => {
      // Une autre sélection a pu intervenir pendant le vol.
      if (selectedRef.current === selectedId) map.openPopup(popup);
    };
    map.stop();
    map.once("moveend", open);
    if (animate) map.flyTo(target, zoom, { duration: 0.8 });
    else map.setView(target, zoom, { animate: false });
    return () => {
      map.off("moveend", open);
    };
  }, [ready, selectedId]);

  const resetView = () => {
    const layers = layersRef.current;
    if (!layers) return;
    layers.map.closePopup();
    if (layers.home) {
      layers.map.flyToBounds(layers.home, {
        padding: [40, 40],
        maxZoom: variant === "public" ? 12 : 14,
        animate: layers.animate,
      });
    } else {
      layers.map.setView(FRANCE_CENTER, FRANCE_ZOOM);
    }
  };

  return (
    <div
      className={cn(
        "relative border-2 border-edge bg-surface shadow-[6px_6px_0_0_var(--color-shadow)]",
        className,
      )}
    >
      <div
        ref={containerRef}
        role="region"
        aria-label={
          variant === "hero"
            ? "Carte des incidents et des super-héros en intervention"
            : "Carte des incidents en cours"
        }
        className="incident-map size-full"
      />
      <button
        type="button"
        onClick={resetView}
        disabled={!ready}
        className="absolute top-2.5 left-2.5 z-10 border-2 border-ink bg-paper px-2.5 py-1.5 font-mono text-[0.6875rem] font-semibold tracking-wider text-ink uppercase shadow-[3px_3px_0_0_var(--color-ink)] hover:bg-mustard disabled:opacity-60"
      >
        Vue d&apos;ensemble
      </button>
      <MapLegend
        showTeams={variant === "hero"}
        className="absolute bottom-2.5 left-2.5 z-10 max-w-[calc(100%-1.25rem)]"
      />
    </div>
  );
}
