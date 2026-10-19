import Link from "next/link";
import { Suspense } from "react";
import { Container } from "@/components/ui/Container";
import { Logo } from "@/components/ui/Logo";
import { mainNav, routes, type NavItem } from "@/lib/routes";
import { NewsletterSection } from "./NewsletterSection";

const footerColumns: Array<{ id: string; title: string; links: NavItem[] }> = [
  {
    id: "footer-explorer",
    title: "Explorer",
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
    <footer className="border-t border-border bg-surface">
      <Container className="grid gap-12 py-12 lg:grid-cols-[1fr_1.1fr] lg:py-16">
        <div className="grid grid-cols-2 gap-8 sm:grid-cols-3">
          {footerColumns.map((column) => (
            <nav key={column.id} aria-labelledby={column.id}>
              <h2 id={column.id} className="font-display text-sm font-semibold">
                {column.title}
              </h2>
              <ul className="mt-3 space-y-1">
                {column.links.map((link) => (
                  <li key={link.href}>
                    <Link
                      href={link.href}
                      className="inline-block rounded-sm py-1 text-sm text-muted hover:text-foreground"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          ))}
        </div>

        <section aria-labelledby="newsletter-title">
          <h2 id="newsletter-title" className="font-display text-lg font-semibold">
            Restez en alerte
          </h2>
          <p className="mt-1 mb-4 text-sm text-muted">
            Une fois par mois : les interventions marquantes, les nouveaux héros et des conseils de
            sécurité.
          </p>
          <Suspense fallback={<div className="min-h-24" aria-hidden="true" />}>
            <NewsletterSection />
          </Suspense>
        </section>
      </Container>

      <div className="border-t border-border">
        <Container className="flex flex-col items-start justify-between gap-3 py-6 text-sm text-muted sm:flex-row sm:items-center">
          <Logo />
          <p>Univers fictif : tous les personnages sont imaginaires.</p>
        </Container>
      </div>
    </footer>
  );
}
