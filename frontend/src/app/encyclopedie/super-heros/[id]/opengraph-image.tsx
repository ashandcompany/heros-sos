import { connection } from "next/server";
import { getSuperHero } from "@/lib/api";
import { dossierNumber } from "@/lib/dossier";
import { formatInteger, formatRating } from "@/lib/format";
import { defaultOgImage, ogContentType, ogImage, ogSize } from "@/lib/og";

export const alt = "Fiche d'un super-héros de l'encyclopédie Heros-SOS";
export const size = ogSize;
export const contentType = ogContentType;

export default async function OpengraphImage({ params }: { params: Promise<{ id: string }> }) {
  // Fiche lue à la requête (API ou mocks), jamais au build.
  await connection();
  const hero = await getSuperHero((await params).id);
  if (!hero) return defaultOgImage();

  const interventions = `${formatInteger(hero.interventionsCount)} intervention${hero.interventionsCount > 1 ? "s" : ""}`;
  return ogImage({
    kicker: "Fiche super-héros",
    title: hero.alias,
    subtitle: `${hero.power} · ${formatRating(hero.averageRating)}/5 · ${interventions}`,
    reference: dossierNumber(hero.id),
  });
}
