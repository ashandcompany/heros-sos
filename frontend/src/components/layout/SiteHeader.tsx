import Link from "next/link";
import { Suspense } from "react";
import { Container } from "@/components/ui/Container";
import { Logo } from "@/components/ui/Logo";
import { routes } from "@/lib/routes";
import { getCurrentUser } from "@/lib/server-session";
import { MainNav } from "./MainNav";

/** Lit la session (cookies) : rendu en streaming, sous <Suspense>. */
async function SessionAwareNav() {
  const user = await getCurrentUser();
  return <MainNav user={user && { displayName: user.displayName, role: user.role }} />;
}

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 border-b border-border/60 bg-background/85 backdrop-blur-md">
      <Container className="relative flex h-16 items-center justify-between gap-6">
        <Link href={routes.home} className="rounded-md" aria-label="Heros-SOS, retour à l'accueil">
          <Logo />
        </Link>
        {/* Pendant la lecture de la session : navigation sans boutons de compte. */}
        <Suspense fallback={<MainNav user={undefined} />}>
          <SessionAwareNav />
        </Suspense>
      </Container>
    </header>
  );
}
