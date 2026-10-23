import type { Metadata } from "next";
import { LoginForm } from "@/components/auth/LoginForm";
import { RefreshSessionNav } from "@/components/auth/RefreshSessionNav";
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
      {/* On arrive souvent ici après l'expiration de la session : la navbar doit le refléter. */}
      <RefreshSessionNav />
      <div className="w-full max-w-md">
        <div className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 sm:p-8">
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
