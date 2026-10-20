import Link from "next/link";
import { Suspense } from "react";
import { Container } from "@/components/ui/Container";
import { DigitalClock } from "@/components/ui/DigitalClock";
import { Logo } from "@/components/ui/Logo";
import { routes } from "@/lib/routes";
import { getCurrentUser, getCurrentUserAvatar } from "@/lib/server-session";
import { MainNav } from "./MainNav";
import { ThemeToggle } from "./ThemeToggle";

/** Lit la session (cookies) : rendu en streaming, sous <Suspense>. */
async function SessionAwareNav() {
  // La session est mise en cache pour la requête : la lire deux fois ne coûte qu'un appel.
  const [user, avatar] = await Promise.all([getCurrentUser(), getCurrentUserAvatar()]);
  return <MainNav user={user && { displayName: user.displayName, role: user.role, avatar }} />;
}

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 bg-background">
      {/* Bandeau officiel */}
      <div className="border-b-2 border-edge bg-surface-raised">
        <Container className="flex h-8 items-center justify-between gap-4 font-mono text-xs tracking-widest text-muted uppercase">
          <p className="truncate">
            <span className="hidden sm:inline">Service de signalement des incidents — </span>
            <span className="sm:hidden">Signalement d&apos;incidents — </span>
            Accès public
          </p>
          <div className="flex shrink-0 items-center gap-3 sm:gap-5">
            <ThemeToggle />
            <DigitalClock />
          </div>
        </Container>
      </div>

      <div className="border-b-4 border-edge">
        <Container className="relative flex h-16 items-center justify-between gap-4">
          <Link
            href={routes.home}
            className="rounded-sm"
            aria-label="Heros-SOS, retour à l'accueil"
          >
            <Logo />
          </Link>
          {/* Pendant la lecture de la session : navigation sans boutons de compte. */}
          <Suspense fallback={<MainNav user={undefined} />}>
            <SessionAwareNav />
          </Suspense>
        </Container>
      </div>
      <div aria-hidden="true" className="bar-tricolor" />
    </header>
  );
}
