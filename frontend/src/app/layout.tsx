import type { Metadata, Viewport } from "next";
import { IBM_Plex_Mono, IBM_Plex_Sans, Jost, Special_Elite } from "next/font/google";
import Script from "next/script";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/SiteFooter";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { SkipLink } from "@/components/layout/SkipLink";
import { OfflineProvider } from "@/components/offline/OfflineProvider";
import { OutboxStatus } from "@/components/offline/OutboxStatus";
import { SessionIdentity } from "@/components/offline/SessionIdentity";
import { themeColors, themeInitScript } from "@/lib/theme";
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
  themeColor: themeColors.dark,
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="fr"
      className={`${jost.variable} ${plexSans.variable} ${plexMono.variable} ${specialElite.variable} h-full`}
      // data-theme est posé par themeInitScript avant l'hydratation.
      suppressHydrationWarning
    >
      <body
        className="flex min-h-full flex-col"
        // Des extensions de navigateur ajoutent des attributs à <body> avant l'hydratation
        // (ex : cz-shortcut-listen de ColorZilla). Ne couvre que <body>, pas ses enfants.
        suppressHydrationWarning
      >
        {/* Injecté dans le HTML initial et exécuté avant l'affichage : pas de flash du thème sombre. */}
        <Script
          id="theme-init"
          strategy="beforeInteractive"
          dangerouslySetInnerHTML={{ __html: themeInitScript }}
        />
        <OfflineProvider>
          <SkipLink />
          <SiteHeader />
          <main id="contenu" tabIndex={-1} className="flex-1 focus:outline-none">
            {children}
          </main>
          <SiteFooter />
          <OutboxStatus />
          {/* Lit la session (cookies) : rendu en streaming, sans bloquer la page. */}
          <Suspense fallback={null}>
            <SessionIdentity />
          </Suspense>
        </OfflineProvider>
      </body>
    </html>
  );
}
