"use client";

import "leaflet/dist/leaflet.css";
import type { LatLngTuple, Map as LeafletMap } from "leaflet";
import { useEffect, useRef } from "react";
import { formatDateTime, incidentPriorityLabels, incidentStatusLabels } from "@/lib/format";
import type { MapTileConfig } from "@/lib/map";
import { heroIncidentRoute } from "@/lib/routes";
import type { IncidentMapData, MapIncident, MapTeam, MapZone } from "@/types";

/** Vue de la France métropolitaine, quand aucun incident n'est localisé. */
const FRANCE_CENTER: LatLngTuple = [46.6, 2.4];
const FRANCE_ZOOM = 6;

/** Bulles : assez larges pour un titre sur deux lignes, sans masquer toute la carte. */
const POPUP_OPTIONS = { minWidth: 240, maxWidth: 300 };

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
}: {
  data: IncidentMapData;
  tiles: MapTileConfig;
  variant?: IncidentMapVariant;
}) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let map: LeafletMap | null = null;
    let cancelled = false;

    void import("leaflet").then(({ default: L }) => {
      if (cancelled) return;

      const animate = !window.matchMedia("(prefers-reduced-motion: reduce)").matches;
      map = L.map(container, {
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

      const incidents = L.layerGroup(
        data.incidents.map((incident) =>
          L.marker([incident.position.lat, incident.position.lng], {
            title: `${incident.title} (${incidentStatusLabels[incident.status]})`,
            riseOnHover: true,
            icon: L.divIcon({
              className: `incident-map__marker incident-map__marker--${incident.priority} incident-map__marker--${incident.status}`,
              html: "<span></span>",
              iconSize: [22, 22],
              iconAnchor: [11, 11],
              popupAnchor: [0, -12],
            }),
          }).bindPopup(() => incidentPopup(incident, variant), POPUP_OPTIONS),
        ),
      );

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

      zones.addTo(map);
      incidents.addTo(map);
      if (variant === "hero") teams.addTo(map);

      L.control
        .layers(
          undefined,
          {
            "Incidents ouverts": incidents,
            ...(variant === "hero" && { "Super-héros en intervention": teams }),
            "Zones touchées": zones,
          },
          { collapsed: false },
        )
        .addTo(map);

      const points: LatLngTuple[] = data.incidents.map(({ position }) => [
        position.lat,
        position.lng,
      ]);
      if (points.length > 0) {
        map.fitBounds(L.latLngBounds(points), {
          padding: [40, 40],
          maxZoom: variant === "public" ? 12 : 14,
        });
      }
    });

    return () => {
      cancelled = true;
      map?.remove();
    };
  }, [data, tiles, variant]);

  return (
    <div
      ref={containerRef}
      role="region"
      aria-label={
        variant === "hero"
          ? "Carte des incidents et des super-héros en intervention"
          : "Carte des incidents en cours"
      }
      className="incident-map h-[min(70vh,560px)] w-full border-2 border-edge bg-surface shadow-[6px_6px_0_0_var(--color-shadow)]"
    />
  );
}
