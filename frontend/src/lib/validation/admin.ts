import { incidentStatusLabels, threatLevelLabels, villainStatusLabels } from "@/lib/format";
import type {
  ArticleInput,
  ArticleStatus,
  IncidentStatus,
  IncidentTypeInput,
  IncidentUpdate,
  ThreatLevel,
  VillainInput,
  VillainStatus,
} from "@/types";
import { DESCRIPTION_MAX_LENGTH, TITLE_MAX_LENGTH, isKnownTypeId } from "./incident";

/* ---------- Correction d'un incident par la modération ---------- */

export const incidentEditFields = [
  "title",
  "type",
  "status",
  "description",
  "address",
  "city",
  "villainId",
  "heroId",
] as const;
export type IncidentEditField = (typeof incidentEditFields)[number];
export type IncidentEditValues = Record<IncidentEditField, string>;

export function readIncidentEditValues(formData: FormData): IncidentEditValues {
  return Object.fromEntries(
    incidentEditFields.map((field) => [field, String(formData.get(field) ?? "").trim()]),
  ) as IncidentEditValues;
}

export function validateIncidentEdit(
  values: IncidentEditValues,
  { villainIds, heroIds, typeIds }: { villainIds: string[]; heroIds: string[]; typeIds: number[] },
): Partial<Record<IncidentEditField, string>> {
  return {
    title:
      values.title.length < 5 || values.title.length > TITLE_MAX_LENGTH
        ? `Le titre doit contenir entre 5 et ${TITLE_MAX_LENGTH} caractères.`
        : undefined,
    type: isKnownTypeId(values.type, typeIds) ? undefined : "Choisissez le type d'incident.",
    status: values.status in incidentStatusLabels ? undefined : "Choisissez un statut.",
    description:
      values.description.length > DESCRIPTION_MAX_LENGTH
        ? `La description ne doit pas dépasser ${DESCRIPTION_MAX_LENGTH} caractères.`
        : undefined,
    address: undefined,
    city: values.city ? undefined : "Indiquez la ville.",
    villainId:
      values.villainId && !villainIds.includes(values.villainId) ? "Vilain inconnu." : undefined,
    heroId:
      values.heroId && !heroIds.includes(values.heroId)
        ? "Super-héros inconnu."
        : values.status && values.status !== "signale" && !values.heroId
          ? "Choisissez le super-héros qui intervient."
          : undefined,
  };
}

export function toIncidentUpdate(values: IncidentEditValues): IncidentUpdate {
  return {
    title: values.title,
    typeId: Number(values.type),
    status: values.status as IncidentStatus,
    description: values.description,
    location: { address: values.address, city: values.city },
    villainId: values.villainId || null,
    heroId: values.status === "signale" ? null : values.heroId || null,
  };
}

/* ---------- Articles du blog ---------- */

export const articleFields = ["title", "excerpt", "content", "status"] as const;
export type ArticleField = (typeof articleFields)[number];

export const articleStatusOptions: Array<{ value: ArticleStatus; label: string }> = [
  { value: "brouillon", label: "Brouillon" },
  { value: "publie", label: "Publié" },
];

export const ARTICLE_TITLE_MAX_LENGTH = 120;
export const ARTICLE_EXCERPT_MAX_LENGTH = 300;
export const ARTICLE_CONTENT_MIN_LENGTH = 20;
const ARTICLE_CONTENT_MAX_BYTES = 200_000;

/** Longueur du texte visible d'un contenu HTML (balises et entités retirées). */
export function visibleTextLength(html: string): number {
  return html
    .replace(/<[^>]*>/g, " ")
    .replace(/&[a-z#0-9]+;/gi, " ")
    .replace(/\s+/g, " ")
    .trim().length;
}

export function readArticleValues(formData: FormData): Record<ArticleField, string> {
  return {
    title: String(formData.get("title") ?? "").trim(),
    excerpt: String(formData.get("excerpt") ?? "").trim(),
    content: String(formData.get("content") ?? ""),
    status: String(formData.get("status") ?? ""),
  };
}

export function validateArticle(
  values: Record<ArticleField, string>,
): Partial<Record<ArticleField, string>> {
  return {
    title:
      values.title.length < 5 || values.title.length > ARTICLE_TITLE_MAX_LENGTH
        ? `Le titre doit contenir entre 5 et ${ARTICLE_TITLE_MAX_LENGTH} caractères.`
        : undefined,
    excerpt: !values.excerpt
      ? "Rédigez un court chapeau."
      : values.excerpt.length > ARTICLE_EXCERPT_MAX_LENGTH
        ? `Le chapeau ne doit pas dépasser ${ARTICLE_EXCERPT_MAX_LENGTH} caractères.`
        : undefined,
    content:
      visibleTextLength(values.content) < ARTICLE_CONTENT_MIN_LENGTH
        ? `Le contenu doit contenir au moins ${ARTICLE_CONTENT_MIN_LENGTH} caractères.`
        : values.content.length > ARTICLE_CONTENT_MAX_BYTES
          ? "Le contenu est trop long."
          : undefined,
    status: articleStatusOptions.some(({ value }) => value === values.status)
      ? undefined
      : "Choisissez un statut.",
  };
}

export function toArticleInput(values: Record<ArticleField, string>): ArticleInput {
  return {
    title: values.title,
    excerpt: values.excerpt,
    content: values.content,
    status: values.status as ArticleStatus,
  };
}

/* ---------- Référentiel des types d'incident ---------- */

export const incidentTypeFields = ["code", "label", "endangersLife"] as const;
export type IncidentTypeField = (typeof incidentTypeFields)[number];
export type IncidentTypeValues = Record<IncidentTypeField, string>;

export const INCIDENT_TYPE_CODE_MAX_LENGTH = 40;
export const INCIDENT_TYPE_LABEL_MAX_LENGTH = 60;
/** Code stable, en minuscules et sans accent : `prise-d-otages`. */
const INCIDENT_TYPE_CODE_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

export function readIncidentTypeValues(formData: FormData): IncidentTypeValues {
  return {
    code: String(formData.get("code") ?? "").trim(),
    label: String(formData.get("label") ?? "").trim(),
    endangersLife: formData.get("endangersLife") === "on" ? "on" : "",
  };
}

export function validateIncidentType(
  values: IncidentTypeValues,
): Partial<Record<IncidentTypeField, string>> {
  return {
    code: !values.code
      ? "Indiquez le code du type."
      : values.code.length > INCIDENT_TYPE_CODE_MAX_LENGTH ||
          !INCIDENT_TYPE_CODE_PATTERN.test(values.code)
        ? `Le code ne contient que des minuscules sans accent, des chiffres et des tirets (${INCIDENT_TYPE_CODE_MAX_LENGTH} caractères au plus).`
        : undefined,
    label: !values.label
      ? "Indiquez le libellé affiché."
      : values.label.length > INCIDENT_TYPE_LABEL_MAX_LENGTH
        ? `Le libellé ne doit pas dépasser ${INCIDENT_TYPE_LABEL_MAX_LENGTH} caractères.`
        : undefined,
    endangersLife: undefined,
  };
}

export function toIncidentTypeInput(values: IncidentTypeValues): IncidentTypeInput {
  return { code: values.code, label: values.label, endangersLife: values.endangersLife === "on" };
}

/* ---------- Fiches des super-vilains ---------- */

export const villainFields = [
  "alias",
  "firstName",
  "lastName",
  "mainSector",
  "description",
  "typeIds",
  "threatLevel",
  "status",
] as const;
export type VillainField = (typeof villainFields)[number];
/** Saisie brute ; `typeIds` : identifiants des spécialités cochées, séparés par des virgules. */
export type VillainValues = Record<VillainField, string>;

export const VILLAIN_ALIAS_MAX_LENGTH = 80;
export const VILLAIN_NAME_MAX_LENGTH = 80;
export const VILLAIN_SECTOR_MAX_LENGTH = 120;
export const VILLAIN_DESCRIPTION_MAX_LENGTH = 2000;

export function readVillainValues(formData: FormData): VillainValues {
  const text = (field: VillainField) => String(formData.get(field) ?? "").trim();
  return {
    alias: text("alias"),
    firstName: text("firstName"),
    lastName: text("lastName"),
    mainSector: text("mainSector"),
    description: text("description"),
    typeIds: formData.getAll("typeIds").map(String).join(","),
    threatLevel: text("threatLevel"),
    status: text("status"),
  };
}

/** Identifiants des spécialités cochées (la saisie brute les sépare par des virgules). */
export function parseVillainTypeIds(typeIds: string): number[] {
  return typeIds ? [...new Set(typeIds.split(",").map(Number))] : [];
}

const tooLong = (value: string, max: number, label: string) =>
  value.length > max ? `${label} ne doit pas dépasser ${max} caractères.` : undefined;

export function validateVillain(
  values: VillainValues,
  { typeIds }: { typeIds: number[] },
): Partial<Record<VillainField, string>> {
  return {
    alias: !values.alias
      ? "Indiquez l'alias du super-vilain."
      : tooLong(values.alias, VILLAIN_ALIAS_MAX_LENGTH, "L'alias"),
    firstName: tooLong(values.firstName, VILLAIN_NAME_MAX_LENGTH, "Le prénom"),
    lastName: tooLong(values.lastName, VILLAIN_NAME_MAX_LENGTH, "Le nom"),
    mainSector: tooLong(values.mainSector, VILLAIN_SECTOR_MAX_LENGTH, "Le secteur"),
    description: tooLong(values.description, VILLAIN_DESCRIPTION_MAX_LENGTH, "La description"),
    typeIds: parseVillainTypeIds(values.typeIds).every((id) => typeIds.includes(id))
      ? undefined
      : "Une spécialité cochée n'existe plus. Rechargez la page.",
    threatLevel: values.threatLevel in threatLevelLabels ? undefined : "Choisissez un niveau.",
    status: values.status in villainStatusLabels ? undefined : "Choisissez un statut.",
  };
}

export function toVillainInput(values: VillainValues): VillainInput {
  return {
    alias: values.alias,
    firstName: values.firstName,
    lastName: values.lastName,
    mainSector: values.mainSector,
    description: values.description,
    typeIds: parseVillainTypeIds(values.typeIds),
    threatLevel: values.threatLevel as ThreatLevel,
    status: values.status as VillainStatus,
  };
}
