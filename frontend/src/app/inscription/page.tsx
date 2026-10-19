import type { Metadata } from "next";
import Link from "next/link";
import { SignupForm } from "@/components/auth/SignupForm";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Inscription",
  description:
    "Créez votre compte Heros-SOS pour signaler des incidents ou intervenir en tant que super-héros.",
};

export default function SignupPage() {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-2xl">
        <div className="rounded-2xl border border-border bg-surface p-6 sm:p-10">
          <h1 className="font-display text-3xl font-bold">Créer un compte</h1>
          <p className="mt-2 mb-8 text-muted">
            Citoyen ou super-héros, l&apos;inscription est la même pour tous. Elle ne prend que deux
            minutes.
          </p>
          <SignupForm />
        </div>

        <p className="mt-6 text-center text-sm text-muted">
          Déjà inscrit·e&nbsp;?{" "}
          <Link href={routes.login} className="rounded-sm font-medium text-accent hover:underline">
            Se connecter
          </Link>
        </p>
      </div>
    </Container>
  );
}
