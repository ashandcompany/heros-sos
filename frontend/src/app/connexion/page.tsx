import type { Metadata } from "next";
import { LoginForm } from "@/components/auth/LoginForm";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Connexion",
  description: "Connectez-vous à Heros-SOS pour signaler un incident ou prendre une mission.",
};

export default function LoginPage() {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-md">
        <div className="rounded-2xl border border-border bg-surface p-6 sm:p-8">
          <h1 className="font-display text-3xl font-bold">Connexion</h1>
          <p className="mt-2 mb-8 text-muted">
            Citoyens et super-héros, accédez à votre espace Heros-SOS.
          </p>
          <LoginForm />
        </div>

        <div className="mt-6 flex flex-col items-center gap-3 text-center">
          <p className="text-sm text-muted">Pas encore de compte&nbsp;?</p>
          <ButtonLink href={routes.signup} variant="secondary" className="w-full">
            Créer un compte
          </ButtonLink>
        </div>
      </div>
    </Container>
  );
}
