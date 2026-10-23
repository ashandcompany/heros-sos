/** Mêmes bornes que l'API (`FilmInput`). */
export const VIDEO_TITLE_MAX_LENGTH = 200;
export const VIDEO_DESCRIPTION_MAX_LENGTH = 4000;
export const VIDEO_URL_MAX_LENGTH = 2048;

export const freeVideoFields = ["title", "description", "videoUrl"] as const;
export type FreeVideoField = (typeof freeVideoFields)[number];
export type FreeVideoValues = Record<FreeVideoField, string>;

export function readFreeVideoValues(formData: FormData): FreeVideoValues {
  return Object.fromEntries(
    freeVideoFields.map((field) => [field, String(formData.get(field) ?? "").trim()]),
  ) as FreeVideoValues;
}

/** Adresse `http` ou `https` avec un hôte, comme l'exige l'API. */
function isVideoUrl(value: string): boolean {
  if (value.length > VIDEO_URL_MAX_LENGTH) return false;
  try {
    const url = new URL(value);
    return (url.protocol === "http:" || url.protocol === "https:") && url.host !== "";
  } catch {
    return false;
  }
}

/** Vidéo libre (hors IMDb) : titre et lien obligatoires, description facultative. */
export function validateFreeVideo(
  values: FreeVideoValues,
): Partial<Record<FreeVideoField, string>> {
  return {
    title: !values.title
      ? "Indiquez le titre de la vidéo."
      : values.title.length > VIDEO_TITLE_MAX_LENGTH
        ? `Le titre ne doit pas dépasser ${VIDEO_TITLE_MAX_LENGTH} caractères.`
        : undefined,
    description:
      values.description.length > VIDEO_DESCRIPTION_MAX_LENGTH
        ? `La description ne doit pas dépasser ${VIDEO_DESCRIPTION_MAX_LENGTH} caractères.`
        : undefined,
    videoUrl: !values.videoUrl
      ? "Indiquez le lien de la vidéo."
      : isVideoUrl(values.videoUrl)
        ? undefined
        : "Indiquez une adresse complète, commençant par https://.",
  };
}
