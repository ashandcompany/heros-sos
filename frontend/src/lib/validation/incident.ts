import type { IncidentDeclaration, ReporterRole } from "@/types";

export const declarationFields = [
  "reporterRole",
  "type",
  "title",
  "description",
  "address",
  "city",
  "occurredAt",
  "villainId",
] as const;
export type DeclarationField = (typeof declarationFields)[number];
export type DeclarationValues = Record<DeclarationField, string>;

export const reporterRoleOptions: Array<{ value: ReporterRole; label: string }> = [
  { value: "temoin", label: "Témoin" },
  { value: "victime", label: "Victime" },
];

export const TITLE_MAX_LENGTH = 80;
export const DESCRIPTION_MIN_LENGTH = 20;
export const DESCRIPTION_MAX_LENGTH = 1000;

const length = (value: string, min: number, max: number, label: string) => {
  if (value.length < min) return `${label} doit contenir au moins ${min} caractères.`;
  if (value.length > max) return `${label} ne doit pas dépasser ${max} caractères.`;
  return undefined;
};

export function readDeclarationValues(formData: FormData): DeclarationValues {
  return Object.fromEntries(
    declarationFields.map((field) => [field, String(formData.get(field) ?? "").trim()]),
  ) as DeclarationValues;
}

/**
 * `occurredAt` est la valeur d'un champ datetime-local, convertie en ISO par le navigateur
 * (fuseau de l'utilisateur) dans le champ caché `occurredAtIso`.
 */
export function validateDeclaration(
  values: DeclarationValues,
  { villainIds, typeIds }: { villainIds: string[]; typeIds: number[] },
): Partial<Record<DeclarationField, string>> {
  return {
    reporterRole: reporterRoleOptions.some((option) => option.value === values.reporterRole)
      ? undefined
      : "Précisez si vous êtes témoin ou victime.",
    type: isKnownTypeId(values.type, typeIds) ? undefined : "Choisissez le type d'incident.",
    title: values.title
      ? length(values.title, 5, TITLE_MAX_LENGTH, "Le titre")
      : "Résumez l'incident en quelques mots.",
    description: values.description
      ? length(values.description, DESCRIPTION_MIN_LENGTH, DESCRIPTION_MAX_LENGTH, "La description")
      : "Décrivez ce qu'il se passe.",
    address: values.address ? undefined : "Indiquez le lieu précis (adresse, repère…).",
    city: values.city ? undefined : "Indiquez la ville.",
    occurredAt:
      values.occurredAt && Number.isNaN(Date.parse(values.occurredAt))
        ? "La date n'est pas valide."
        : values.occurredAt && Date.parse(values.occurredAt) > Date.now() + 60_000
          ? "La date ne peut pas être dans le futur."
          : undefined,
    villainId:
      values.villainId && !villainIds.includes(values.villainId) ? "Vilain inconnu." : undefined,
  };
}

export function toDeclaration(values: DeclarationValues): IncidentDeclaration {
  return {
    reporterRole: (values.reporterRole as ReporterRole) || null,
    typeId: Number(values.type),
    title: values.title,
    description: values.description,
    location: { address: values.address, city: values.city },
    occurredAt: values.occurredAt ? new Date(values.occurredAt).toISOString() : null,
    villainId: values.villainId || null,
  };
}

/** Le champ `type` porte l'identifiant d'un type du référentiel (`IncidentTypeDefinition.id`). */
export function isKnownTypeId(value: string, typeIds: number[]): boolean {
  return /^\d+$/.test(value) && typeIds.includes(Number(value));
}
