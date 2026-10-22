import { connection } from "next/server";
import { getVillain } from "@/lib/api";
import { dossierNumber } from "@/lib/dossier";
import { threatLevelLabels, villainStatusLabels } from "@/lib/format";
import { defaultOgImage, ogContentType, ogImage, ogSize } from "@/lib/og";

export const alt = "Fiche d'un super-vilain de l'encyclopédie Heros-SOS";
export const size = ogSize;
export const contentType = ogContentType;

export default async function OpengraphImage({ params }: { params: Promise<{ id: string }> }) {
  // Fiche lue à la requête (API ou mocks), jamais au build.
  await connection();
  const villain = await getVillain((await params).id);
  if (!villain) return defaultOgImage();

  return ogImage({
    kicker: "Fiche super-vilain",
    title: villain.alias,
    subtitle: `Menace ${threatLevelLabels[villain.threatLevel].toLowerCase()} · ${villainStatusLabels[villain.status]}`,
    reference: dossierNumber(villain.id, "VIL"),
  });
}
