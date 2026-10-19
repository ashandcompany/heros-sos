import type { Metadata } from "next";
import { SpaceNav } from "@/components/layout/SpaceNav";
import { Container } from "@/components/ui/Container";
import { citizenNav } from "@/lib/routes";

export const metadata: Metadata = {
  title: { template: "%s · Mon espace · Heros-SOS", default: "Mon espace" },
  robots: { index: false, follow: false },
};

/** Espace personnel des citoyens, réservé au rôle « citoyen » (src/proxy.ts + chaque page). */
export default function CitizenSpaceLayout({ children }: LayoutProps<"/profil">) {
  return (
    <Container className="py-10 sm:py-14">
      <p className="mb-4 text-sm font-semibold tracking-widest text-accent uppercase">
        Espace personnel
      </p>
      <SpaceNav items={citizenNav} label="Espace personnel" />
      {children}
    </Container>
  );
}
