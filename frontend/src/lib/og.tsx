import { ImageResponse } from "next/og";

/*
 * Images Open Graph (aperçus des liens partagés) : un dossier papier sur le fond sombre du site,
 * avec le tampon Heros-SOS. Mêmes couleurs que les design tokens de globals.css ; polices par
 * défaut de `ImageResponse`, pour ne rien télécharger au rendu.
 */

export const ogSize = { width: 1200, height: 630 };
export const ogContentType = "image/png";

const colors = {
  background: "#111315",
  paper: "#f2ebd9",
  ink: "#111315",
  primary: "#cf6a3c",
  primaryInk: "#a8330a",
  accent: "#0a6b63",
  mustard: "#f2b134",
};

interface OgCard {
  /** Rubrique, en capitales au-dessus du titre (« Blog », « Fiche super-vilain »…). */
  kicker: string;
  title: string;
  /** Ligne sous le titre : date, pouvoir, niveau de menace… */
  subtitle?: string;
  /** Référence en haut à droite, façon numéro de dossier. */
  reference?: string;
}

export function ogImage({ kicker, title, subtitle, reference }: OgCard): ImageResponse {
  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        padding: 48,
        background: colors.background,
      }}
    >
      <div
        style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          padding: "48px 56px",
          background: colors.paper,
          color: colors.ink,
          border: `6px solid ${colors.ink}`,
          boxShadow: `14px 14px 0 0 ${colors.primary}`,
        }}
      >
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div
            style={{
              display: "flex",
              fontSize: 26,
              fontWeight: 700,
              letterSpacing: 6,
              textTransform: "uppercase",
              color: colors.accent,
            }}
          >
            {kicker}
          </div>
          {reference && (
            <div style={{ display: "flex", fontSize: 24, letterSpacing: 2 }}>{reference}</div>
          )}
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          <div
            style={{
              display: "flex",
              fontSize: title.length > 60 ? 56 : 72,
              fontWeight: 700,
              lineHeight: 1.1,
              textTransform: "uppercase",
              letterSpacing: 2,
            }}
          >
            {title}
          </div>
          {subtitle && (
            <div style={{ display: "flex", fontSize: 32, color: "#3d4248" }}>{subtitle}</div>
          )}
        </div>

        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-end" }}>
          {/* Bandes tricolores du site. */}
          <div style={{ display: "flex", gap: 8 }}>
            {[colors.primary, colors.mustard, colors.accent].map((color) => (
              <div key={color} style={{ width: 64, height: 14, background: color }} />
            ))}
          </div>
          <div
            style={{
              display: "flex",
              padding: "8px 24px",
              border: `5px solid ${colors.primaryInk}`,
              color: colors.primaryInk,
              fontSize: 36,
              fontWeight: 700,
              letterSpacing: 4,
              textTransform: "uppercase",
              transform: "rotate(-4deg)",
            }}
          >
            Heros-SOS
          </div>
        </div>
      </div>
    </div>,
    ogSize,
  );
}

/** Image par défaut du site, aussi utilisée quand l'article ou la fiche est introuvable. */
export const defaultOgImage = () =>
  ogImage({
    kicker: "Signalez, un super-héros intervient",
    title: "Heros-SOS",
    subtitle:
      "Les citoyens signalent les méfaits des super-vilains, les super-héros interviennent.",
  });
