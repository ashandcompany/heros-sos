import type { Metadata } from "next";
import { SpaceNav } from "@/components/layout/SpaceNav";
import { Container } from "@/components/ui/Container";
import { adminNav } from "@/lib/routes";

export const metadata: Metadata = {
  title: { template: "%s · Back-office · Heros-SOS", default: "Back-office" },
  robots: { index: false, follow: false },
};

/** Back-office des modérateurs, réservé au rôle « admin » (src/proxy.ts + chaque composant). */
export default function AdminLayout({ children }: LayoutProps<"/admin">) {
  return (
    <Container className="py-10 sm:py-14">
      <p className="mb-4 text-sm font-semibold tracking-widest text-accent uppercase">
        Back-office · Modération
      </p>
      <SpaceNav items={adminNav} label="Back-office" />
      {children}
    </Container>
  );
}
