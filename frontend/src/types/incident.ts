import type { ApiFailure } from "./api";
import type { ModerationStatus } from "./moderation";
import type { SuperHeroSummary } from "./super-hero";
import type { VillainSummary } from "./villain";

/**
 * Type d'un incident, tel qu'il est embarqué dans l'incident (`typeCode` et `typeLabel` de
 * l'API). Les types forment un référentiel géré par la modération : aucune liste n'est écrite en dur.
 */
export interface IncidentType {
  /** Code stable (`vol`, `prise-d-otages`…). */
  code: string;
  /** Libellé affiché. */
  label: string;
}

/** Type du référentiel (`GET /intervention-types`), proposé dans les formulaires. */
export interface IncidentTypeDefinition extends IncidentType {
  /** Identifiant envoyé à l'API (`typeId`). */
  id: number;
  /** Le type d'incident met des vies en danger. */
  endangersLife: boolean;
}

/** Saisie d'un nouveau type par la modération (`InterventionTypePayload`). */
export interface IncidentTypeInput {
  code: string;
  label: string;
  endangersLife: boolean;
}

export type IncidentTypeCreationResult =
  | { ok: true; type: IncidentTypeDefinition }
  | { ok: false; error: "code-already-used" }
  | { ok: false; error: "invalid-fields"; fields: Record<string, string> }
  | ApiFailure;

export type IncidentStatus = "signale" | "en-cours" | "resolu";

/** Urgence de l'incident, définie par les super-héros. */
export type IncidentPriority = "basse" | "normale" | "haute" | "critique";

/** Résultat d'une intervention terminée. */
export type InterventionOutcome = "reussi" | "echoue";

/** Le citoyen qui déclare est témoin ou victime de l'incident. */
export type ReporterRole = "temoin" | "victime";

/** Coordonnées WGS 84 (degrés décimaux). */
export interface GeoPoint {
  lat: number;
  lng: number;
}

export interface IncidentLocation {
  city: string;
  district?: string;
  /** Lieu précis saisi par le déclarant (adresse, repère…). */
  address?: string;
  /**
   * Position géocodée à partir de l'adresse et de la ville (Base Adresse Nationale).
   * Absente si le lieu n'a pas pu être géocodé : l'incident n'apparaît alors pas sur la carte.
   */
  coordinates?: GeoPoint;
}

export interface Incident {
  id: string;
  title: string;
  type: IncidentType;
  status: IncidentStatus;
  /**
   * Vérification par la modération (déclaration conforme et réelle).
   * Un incident rejeté n'apparaît plus dans les listes publiques.
   */
  moderation: ModerationStatus;
  /** Priorité fixée par un super-héros (`normale` tant qu'elle n'a pas été définie). */
  priority?: IncidentPriority;
  location: IncidentLocation;
  /** Date ISO 8601 du signalement. */
  reportedAt: string;
  /** Date ISO 8601 de résolution, présente uniquement si `status === "resolu"`. */
  resolvedAt?: string;
  /** Vilain impliqué, `null` s'il n'a pas été identifié. */
  villain: VillainSummary | null;
  /** Super-héros intervenu, absent tant que l'incident n'est pas pris en charge. */
  hero?: SuperHeroSummary;
  description?: string;
  /** Date ISO 8601 des faits si elle diffère du signalement. */
  occurredAt?: string;
  /** Citoyen ayant déclaré l'incident. */
  reporterId?: string;
  reporterRole?: ReporterRole;
}

/** Incident traité avec succès : un héros est forcément intervenu. */
export type ResolvedIncident = Incident & {
  status: "resolu";
  resolvedAt: string;
  hero: SuperHeroSummary;
};

/** Données saisies par un citoyen pour déclarer un incident. */
export interface IncidentDeclaration {
  /** Identifiant du type dans le référentiel (`typeId` de l'API). */
  typeId: number;
  title: string;
  description: string;
  reporterRole: ReporterRole;
  location: { address: string; city: string };
  /** Date ISO 8601 des faits, `null` si l'incident est en cours. */
  occurredAt: string | null;
  /** `null` si le citoyen ne sait pas quel vilain est impliqué. */
  villainId: string | null;
}

/** Évaluation d'une intervention par le citoyen impliqué. */
export interface InterventionRating {
  incidentId: string;
  heroId: string;
  /** Note entière de 1 à 5. */
  rating: number;
  comment: string | null;
  /** Modération du commentaire : un commentaire rejeté est masqué (la note reste comptée). */
  commentModeration: ModerationStatus;
  ratedAt: string;
  /** Réponse du super-héros au commentaire. */
  reply?: HeroReply | null;
}

export interface HeroReply {
  content: string;
  /** Date ISO 8601. */
  repliedAt: string;
}

/** Évaluation affichée dans le back-office, avec l'incident et le héros concernés. */
export interface RatingForModeration extends InterventionRating {
  incident: Pick<Incident, "id" | "title">;
  hero: SuperHeroSummary;
}

/** Modifications d'un incident par la modération. */
export interface IncidentUpdate {
  title: string;
  /** Identifiant du type dans le référentiel. */
  typeId: number;
  status: IncidentStatus;
  description: string;
  location: { address: string; city: string };
  villainId: string | null;
  /** Obligatoire dès que l'incident est pris en charge. */
  heroId: string | null;
}

/** Intervention terminée sur un incident du citoyen, avec sa note s'il l'a déjà donnée. */
export interface RateableIntervention {
  incident: ResolvedIncident;
  rating: InterventionRating | null;
}

/**
 * Intervention d'un super-héros sur un incident. Un seul héros intervient à la fois :
 * après un échec, l'incident redevient disponible pour une nouvelle prise en charge.
 */
export interface HeroIntervention {
  id: string;
  incidentId: string;
  hero: SuperHeroSummary;
  /** Dates ISO 8601. */
  startedAt: string;
  endedAt: string | null;
  /** `null` tant que l'intervention est en cours. */
  outcome: InterventionOutcome | null;
}

/** Incident vu depuis l'espace super-héros : priorité et historique des interventions. */
export interface HeroIncident extends Incident {
  priority: IncidentPriority;
  /** Interventions sur cet incident, des plus récentes aux plus anciennes. */
  interventions: HeroIntervention[];
}

/**
 * Avis affiché sur la fiche publique d'un super-héros : uniquement les commentaires publiés
 * par la modération, sans aucune information sur le citoyen qui l'a écrit.
 */
export interface PublicReview {
  incidentId: string;
  rating: number;
  comment: string;
  ratedAt: string;
  reply: HeroReply | null;
}

/** Détail d'un incident consulté depuis l'espace citoyen. */
export interface CitizenIncidentDetail {
  /** Sans `reporterId` quand l'incident a été déclaré par quelqu'un d'autre. */
  incident: Incident;
  /** L'incident a été déclaré par le citoyen connecté. */
  isMine: boolean;
  /** Interventions des super-héros, des plus anciennes aux plus récentes (en cours comprise). */
  interventions: HeroIntervention[];
  /** Note déjà donnée par le citoyen (uniquement pour ses propres incidents). */
  rating: InterventionRating | null;
}

/** Incident placé sur la carte des super-héros : uniquement ce qu'il faut pour l'afficher. */
export interface MapIncident {
  id: string;
  title: string;
  type: IncidentType;
  status: Exclude<IncidentStatus, "resolu">;
  priority: IncidentPriority;
  /** Ville et quartier, déjà mis en forme. */
  place: string;
  position: GeoPoint;
  /** Date ISO 8601 du signalement. */
  reportedAt: string;
}

/**
 * Super-héros en intervention. Sa position est celle de l'incident qu'il traite : la plateforme
 * ne suit pas la géolocalisation des héros en temps réel.
 */
export interface MapTeam {
  hero: SuperHeroSummary;
  incidentId: string;
  position: GeoPoint;
  /** Date ISO 8601 du début de l'intervention, `null` si elle n'est pas connue. */
  startedAt: string | null;
}

/** Zone touchée : regroupement des incidents non résolus d'une même ville. */
export interface MapZone {
  city: string;
  /** Centre des incidents de la zone. */
  center: GeoPoint;
  /** Distance en mètres entre le centre et l'incident le plus éloigné. */
  radius: number;
  incidentsCount: number;
}

export interface IncidentMapData {
  incidents: MapIncident[];
  teams: MapTeam[];
  zones: MapZone[];
  /** Incidents non résolus dont le lieu n'a pas pu être géocodé. */
  unlocatedCount: number;
}
