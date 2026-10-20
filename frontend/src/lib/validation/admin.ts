import { incidentStatusLabels } from "@/lib/format";
import type {
  ArticleInput,
  ArticleStatus,
  IncidentStatus,
  IncidentTypeInput,
  IncidentUpdate,
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
