import type { Metadata } from "next";
import { Container } from "@/components/ui/Container";

export const metadata: Metadata = {
  title: "Hors ligne",
  robots: { index: false, follow: false },
};

/**
 * Page de repli du service worker (public/sw.js) : affichée quand une page n'a jamais été chargée sur cet
 * appareil et que le réseau manque. Elle est gardée en cache dès l'installation du service worker.
 */
export default function OfflinePage() {
  return (
    <Container className="py-14">
      <h1 className="font-display text-3xl tracking-widest uppercase">Hors ligne</h1>
      <p className="mt-4 max-w-prose text-muted">
        Cette page n&apos;a pas encore été chargée sur cet appareil, et le réseau manque. Les pages
        déjà ouvertes restent disponibles, et vos actions (déclarer un incident, prendre en charge
        une intervention…) sont gardées sur l&apos;appareil : elles partiront dès le retour du
        réseau.
      </p>
    </Container>
  );
}
