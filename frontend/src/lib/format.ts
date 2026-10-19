import type {
  IncidentLocation,
  IncidentStatus,
  IncidentType,
  ModerationStatus,
  ReporterRole,
  ThreatLevel,
} from "@/types";

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

const integerFormatter = new Intl.NumberFormat("fr-FR");

export function formatInteger(value: number): string {
  return integerFormatter.format(value);
}

/** « 06 12 34 56 78 », « +33 6 12 34 56 78 », sinon le numéro tel quel. */
export function formatPhone(phone: string): string {
  if (/^0\d{9}$/.test(phone)) return phone.replace(/(\d{2})(?=\d)/g, "$1 ");
  const french = phone.match(/^\+33(\d)(\d{8})$/);
  if (french) return `+33 ${french[1]} ${french[2].replace(/(\d{2})(?=\d)/g, "$1 ")}`;
  return phone;
}

/** Valeur de lien `tel:` au format international quand c'est possible. */
export function phoneHref(phone: string): string {
  return `tel:${/^0\d{9}$/.test(phone) ? `+33${phone.slice(1)}` : phone}`;
}

/** « NOM Prénom », ou `null` si aucun des deux n'est connu. */
export function formatFullName(lastName: string | null, firstName: string | null): string | null {
  const parts = [lastName?.toLocaleUpperCase("fr-FR"), firstName].filter(Boolean);
  return parts.length > 0 ? parts.join(" ") : null;
}

const dateTimeFormatter = new Intl.DateTimeFormat("fr-FR", {
  day: "numeric",
  month: "long",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "Europe/Paris",
});

export function formatDateTime(isoDate: string): string {
  return dateTimeFormatter.format(new Date(isoDate));
}

export const incidentStatusLabels: Record<IncidentStatus, string> = {
  signale: "Signalé",
  "en-cours": "Intervention en cours",
  resolu: "Résolu",
};

export const reporterRoleLabels: Record<ReporterRole, string> = {
  temoin: "Témoin",
  victime: "Victime",
};

export const threatLevelLabels: Record<ThreatLevel, string> = {
  faible: "Faible",
  modere: "Modérée",
  eleve: "Élevée",
  critique: "Critique",
};

export const moderationStatusLabels: Record<ModerationStatus, string> = {
  "en-attente": "À vérifier",
  valide: "Validé",
  rejete: "Rejeté",
};

export const isModerationStatus = (value: string): value is ModerationStatus =>
  value in moderationStatusLabels;
