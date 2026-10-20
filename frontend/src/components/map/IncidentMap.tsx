"use client";

import "leaflet/dist/leaflet.css";
import type { LatLngTuple, Map as LeafletMap } from "leaflet";
import { useEffect, useRef } from "react";
import {
  formatDateTime,
  incidentPriorityLabels,
  incidentStatusLabels,
  incidentTypeLabels,
} from "@/lib/format";
import type { MapTileConfig } from "@/lib/map";
import { heroIncidentRoute } from "@/lib/routes";
import type { IncidentMapData, MapIncident, MapTeam, MapZone } from "@/types";

/** Vue de la France métropolitaine, quand aucun incident n'est localisé. */
const FRANCE_CENTER: LatLngTuple = [46.6, 2.4];
const FRANCE_ZOOM = 6;

/**
 * Contenu d'une bulle, construit nœud par nœud : titres et lieux viennent des citoyens,
 * ils ne doivent jamais être interprétés comme du HTML.
 */
function popupContent(lines: Array<string | null>, link?: { href: string; label: string }) {
  const root = document.createElement("div");
  root.className = "incident-map__popup";
  lines.forEach((text, index) => {
    if (!text) return;
    const line = document.createElement(index === 0 ? "strong" : "p");
    line.textContent = text;
    root.append(line);
  });
  if (link) {
    const anchor = document.createElement("a");
    anchor.href = link.href;
    anchor.textContent = link.label;
    root.append(anchor);
  }
  return root;
}

const incidentPopup = (incident: MapIncident) =>
  popupContent(
    [
      incident.title,
      `${incidentTypeLabels[incident.type]} · ${incidentStatusLabels[incident.status]} · priorité ${incidentPriorityLabels[incident.priority].toLowerCase()}`,
      incident.place,
      `Signalé le ${formatDateTime(incident.reportedAt)}`,
    ],
    { href: heroIncidentRoute(incident.id), label: "Voir l'incident" },
  );

const teamPopup = (team: MapTeam) =>
  popupContent(
    [
      team.hero.alias,
      "En intervention",
      team.startedAt ? `Depuis le ${formatDateTime(team.startedAt)}` : null,
    ],
    { href: heroIncidentRoute(team.incidentId), label: "Voir l'intervention" },
  );

const zonePopup = (zone: MapZone) =>
  popupContent([
    zone.city,
    `${zone.incidentsCount} incident${zone.incidentsCount > 1 ? "s" : ""} non résolu${zone.incidentsCount > 1 ? "s" : ""}`,
  ]);

/**
 * Carte Leaflet des incidents non résolus, des super-héros en intervention et des zones touchées.
 * Leaflet manipule `window` dès son chargement : il est importé dans l'effet, côté navigateur.
 * La carte n'est pas accessible à tous : la page affiche aussi la liste des incidents.
 */
export function IncidentMap({ data, tiles }: { data: IncidentMapData; tiles: MapTileConfig }) {
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
          }).bindPopup(() => zonePopup(zone)),
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
          }).bindPopup(() => incidentPopup(incident)),
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
          }).bindPopup(() => teamPopup(team)),
        ),
      );

      zones.addTo(map);
      incidents.addTo(map);
      teams.addTo(map);

      L.control
        .layers(
          undefined,
          {
            "Incidents ouverts": incidents,
            "Super-héros en intervention": teams,
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
        map.fitBounds(L.latLngBounds(points), { padding: [40, 40], maxZoom: 14 });
      }
    });

    return () => {
      cancelled = true;
      map?.remove();
    };
  }, [data, tiles]);

  return (
    <div
      ref={containerRef}
      role="region"
      aria-label="Carte des incidents et des super-héros en intervention"
      className="incident-map h-[min(70vh,560px)] w-full border-2 border-edge bg-surface shadow-[6px_6px_0_0_var(--color-shadow)]"
    />
  );
}
