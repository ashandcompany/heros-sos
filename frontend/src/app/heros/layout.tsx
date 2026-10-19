import type { Metadata } from "next";
import { SpaceNav } from "@/components/layout/SpaceNav";
import { Container } from "@/components/ui/Container";
import { heroNav } from "@/lib/routes";

export const metadata: Metadata = {
  title: { template: "%s · Espace pro · Heros-SOS", default: "Espace professionnel" },
  robots: { index: false, follow: false },
};

/**
 * Espace professionnel des super-héros, réservé au rôle « super-heros » (src/proxy.ts + chaque
 * composant). Un compte super-héros n'existe qu'après validation de la demande par la modération.
 */
export default function HeroSpaceLayout({ children }: LayoutProps<"/heros">) {
  return (
    <Container className="py-10 sm:py-14">
      <p className="mb-4 text-sm font-semibold tracking-widest text-accent uppercase">
        Espace professionnel · Super-héros
      </p>
      <SpaceNav items={heroNav} label="Espace professionnel" />
      {children}
    </Container>
  );
}
