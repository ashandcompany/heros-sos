import type { Metadata, Viewport } from "next";
import { Inter, Sora } from "next/font/google";
import { SiteFooter } from "@/components/layout/SiteFooter";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { SkipLink } from "@/components/layout/SkipLink";
import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const sora = Sora({
  variable: "--font-sora",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: {
    default: "Heros-SOS — Signalez, un super-héros intervient",
    template: "%s · Heros-SOS",
  },
  description:
    "Heros-SOS met en relation les citoyens et les super-héros : signalez un incident causé par un super-vilain et suivez l'intervention en temps réel.",
};

export const viewport: Viewport = {
  themeColor: "#070910",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="fr" className={`${inter.variable} ${sora.variable} h-full`}>
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
