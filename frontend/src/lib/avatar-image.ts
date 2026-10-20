/*
 * Préparation de la photo de profil dans le navigateur, avant l'envoi : recadrage carré,
 * redimensionnement et compression. L'image envoyée pèse quelques dizaines de Ko au lieu de
 * plusieurs Mo, ce qui permet de l'afficher partout (y compris dans la navigation) sans alourdir
 * les pages.
 */

/** Côté de l'image envoyée, en pixels : assez pour un affichage net en 80 px sur écran 3x. */
export const AVATAR_OUTPUT_SIZE = 256;

/** Zone à garder, en pixels de l'image d'origine (format de `react-easy-crop`). */
export interface CropArea {
  x: number;
  y: number;
  width: number;
  height: number;
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("Image illisible"));
    image.src = src;
  });
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string): Promise<Blob | null> {
  return new Promise((resolve) => canvas.toBlob(resolve, type, 0.85));
}

/**
 * Découpe `area` dans l'image `src` et la renvoie en carré de `AVATAR_OUTPUT_SIZE` pixels,
 * en WebP (ou en JPEG pour les navigateurs qui ne savent pas encoder le WebP).
 */
export async function cropAvatar(src: string, area: CropArea): Promise<File> {
  const image = await loadImage(src);
  const canvas = document.createElement("canvas");
  canvas.width = AVATAR_OUTPUT_SIZE;
  canvas.height = AVATAR_OUTPUT_SIZE;

  const context = canvas.getContext("2d");
  if (!context) throw new Error("Canvas indisponible");
  context.imageSmoothingQuality = "high";
  context.drawImage(
    image,
    area.x,
    area.y,
    area.width,
    area.height,
    0,
    0,
    AVATAR_OUTPUT_SIZE,
    AVATAR_OUTPUT_SIZE,
  );

  // Un navigateur qui ne sait pas encoder le format demandé renvoie du PNG : on se rabat sur JPEG.
  let blob = await canvasToBlob(canvas, "image/webp");
  if (blob?.type !== "image/webp") blob = await canvasToBlob(canvas, "image/jpeg");
  if (!blob) throw new Error("Encodage impossible");

  const extension = blob.type === "image/webp" ? "webp" : "jpg";
  return new File([blob], `avatar.${extension}`, { type: blob.type });
}
