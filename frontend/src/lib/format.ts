import type { IncidentLocation, IncidentType } from "@/types";

const dateFormatter = new Intl.DateTimeFormat("fr-FR", {
  day: "numeric",
  month: "long",
  year: "numeric",
  timeZone: "Europe/Paris",
});

const ratingFormatter = new Intl.NumberFormat("fr-FR", {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
});

export function formatDate(isoDate: string): string {
  return dateFormatter.format(new Date(isoDate));
}

export function formatRating(rating: number): string {
  return ratingFormatter.format(rating);
}

export function formatLocation({ city, district }: IncidentLocation): string {
  return district ? `${city} · ${district}` : city;
}

export const incidentTypeLabels: Record<IncidentType, string> = {
  vol: "Vol",
  cambriolage: "Cambriolage",
  "prise-d-otages": "Prise d'otages",
  agression: "Agression",
  sabotage: "Sabotage",
  enlevement: "Enlèvement",
};
