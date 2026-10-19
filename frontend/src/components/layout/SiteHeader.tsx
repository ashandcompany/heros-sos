import Link from "next/link";
import { Container } from "@/components/ui/Container";
import { Logo } from "@/components/ui/Logo";
import { routes } from "@/lib/routes";
import { MainNav } from "./MainNav";

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 border-b border-border/60 bg-background/85 backdrop-blur-md">
      <Container className="relative flex h-16 items-center justify-between gap-6">
        <Link href={routes.home} className="rounded-md" aria-label="Heros-SOS, retour à l'accueil">
          <Logo />
        </Link>
        <MainNav />
      </Container>
    </header>
  );
}
