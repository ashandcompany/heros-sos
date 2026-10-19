import Link from "next/link";
import { Suspense } from "react";
import { Container } from "@/components/ui/Container";
import { GeneratedOn } from "@/components/ui/DigitalClock";
import { Logo } from "@/components/ui/Logo";
import { mainNav, routes, type NavItem } from "@/lib/routes";
import { NewsletterSection } from "./NewsletterSection";

const footerColumns: Array<{ id: string; title: string; links: NavItem[] }> = [
  {
    id: "footer-explorer",
    title: "Rubriques",
    links: mainNav.filter((item) => item.href !== routes.home),
  },
  {
    id: "footer-plateforme",
    title: "Heros-SOS",
    links: [
      { label: "À propos", href: routes.about },
      { label: "Questions fréquentes", href: routes.faq },
      { label: "Contact", href: routes.contact },
    ],
  },
  {
    id: "footer-informations",
    title: "Informations",
    links: [
      { label: "Mentions légales", href: routes.legal },
      { label: "Confidentialité", href: routes.privacy },
    ],
  },
];

export function SiteFooter() {
  return (
    <footer className="bg-wood border-t-8 border-double border-edge text-foreground">
      <Container className="grid gap-12 py-12 lg:grid-cols-[1fr_1.1fr] lg:py-16">
        <div className="grid grid-cols-2 gap-8 sm:grid-cols-3">
          {footerColumns.map((column) => (
            <nav key={column.id} aria-labelledby={column.id}>
              <h2
                id={column.id}
                className="border-b-2 border-border pb-1 font-mono text-xs font-medium tracking-widest text-accent uppercase"
              >
                {column.title}
              </h2>
              <ul className="mt-3 space-y-1">
                {column.links.map((link) => (
                  <li key={link.href}>
                    <Link
                      href={link.href}
                      className="inline-block rounded-sm py-1 text-sm text-muted hover:text-foreground hover:underline"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          ))}
        </div>

        <section aria-labelledby="newsletter-title" className="panel rivets paper p-5 sm:p-6">
          <p aria-hidden="true" className="font-mono text-xs tracking-widest text-ink/75 uppercase">
            Formulaire n° 12-C
          </p>
          <h2
            id="newsletter-title"
            className="mt-1 font-display text-lg font-bold tracking-wider uppercase"
          >
            Demande d&apos;abonnement au bulletin officiel
          </h2>
          <p className="mt-2 mb-5 text-sm text-pretty text-ink/85">
            Une fois par mois : interventions marquantes, nouveaux agents et recommandations de
            sécurité. Veuillez remplir le champ ci-dessous, puis apposer le tampon.
          </p>
          <Suspense fallback={<div className="min-h-24" aria-hidden="true" />}>
            <NewsletterSection />
          </Suspense>
        </section>
      </Container>

      <div className="border-t-4 border-edge bg-background">
        <Container className="flex flex-col items-start justify-between gap-4 py-6 font-mono text-xs tracking-wider text-muted uppercase sm:flex-row sm:items-center">
          <Logo />
          <div className="space-y-1 sm:text-right">
            <p>Univers fictif : tous les personnages sont imaginaires.</p>
            <p>
              Formulaire HSOS-7B (rév. 1977) — Document généré le <GeneratedOn />
            </p>
          </div>
        </Container>
      </div>
    </footer>
  );
}
