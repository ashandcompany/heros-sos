import type { Metadata } from "next";
import { Suspense } from "react";
import {
  SuperHeroesDirectory,
  SuperHeroesDirectorySkeleton,
} from "@/components/admin/SuperHeroesDirectory";
import { Container } from "@/components/ui/Container";

export const metadata: Metadata = {
  title: "Super-héros inscrits",
  robots: { index: false, follow: false },
};

/** Back-office, réservé aux administrateurs (src/proxy.ts + vérification dans le composant). */
export default function AdminSuperHeroesPage() {
  return (
    <Container className="py-10 sm:py-14">
      <p className="mb-2 text-sm font-semibold tracking-widest text-accent uppercase">
        Back-office
      </p>
      <h1 className="font-display text-3xl font-bold">Super-héros inscrits</h1>
      <p className="mt-2 mb-8 max-w-2xl text-muted">
        Référentiel de tous les super-héros de la plateforme. Ces informations sont
        confidentielles&nbsp;: ne les partagez pas en dehors de l&apos;équipe.
      </p>
      <Suspense fallback={<SuperHeroesDirectorySkeleton />}>
        <SuperHeroesDirectory />
      </Suspense>
    </Container>
  );
}
