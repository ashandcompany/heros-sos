import type { Metadata, Viewport } from "next";
import { IBM_Plex_Mono, IBM_Plex_Sans, Jost, Special_Elite } from "next/font/google";
import { SiteFooter } from "@/components/layout/SiteFooter";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { SkipLink } from "@/components/layout/SkipLink";
import "./globals.css";
import "./retro.css";

// Titres : clone libre de Futura, en capitales espacées.
const jost = Jost({
  variable: "--font-jost",
  subsets: ["latin"],
});

// Texte courant : sobre, corporate.
const plexSans = IBM_Plex_Sans({
  variable: "--font-plex-sans",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

// Données : numéros d'incident, dates, statuts.
const plexMono = IBM_Plex_Mono({
  variable: "--font-plex-mono",
  subsets: ["latin"],
  weight: ["400", "500", "600"],
});

// Réservée aux tampons.
const specialElite = Special_Elite({
  variable: "--font-special-elite",
  subsets: ["latin"],
  weight: "400",
});

export const metadata: Metadata = {
  title: {
    default: "Heros-SOS · Signalez, un super-héros intervient",
    template: "%s · Heros-SOS",
  },
  description:
    "Heros-SOS met en relation les citoyens et les super-héros : signalez un incident causé par un super-vilain et suivez l'intervention en temps réel.",
};

export const viewport: Viewport = {
  themeColor: "#1e1611",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="fr" className={`${jost.variable} ${plexSans.variable} ${plexMono.variable} ${specialElite.variable} h-full`}>
      <body className="flex min-h-full flex-col">
        <SkipLink />
        <SiteHeader />
        <main id="contenu" tabIndex={-1} className="flex-1 focus:outline-none">
          {children}
        </main>
        <SiteFooter />
      </body>
    </html>
  );
}
