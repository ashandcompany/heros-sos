import { Panel } from "@/components/citizen/AccountDetails";
import { MfaSetup } from "@/components/auth/MfaSetup";
import { RegenerateRecoveryCodesForm } from "@/components/auth/RegenerateRecoveryCodesForm";
import { Badge } from "@/components/ui/Badge";
import { getMfaOverview } from "@/lib/api";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";

/** Seuil à partir duquel il faut penser à renouveler les codes de secours. */
const LOW_RECOVERY_CODES = 3;

/** Double authentification du modérateur connecté. Lit la session : à rendre sous <Suspense>. */
export async function MfaPanel() {
  const token = await requireAdminSession();
  const { status, recoveryCodesRemaining } = await getMfaOverview(token);

  if (status === "verified") {
    const low = recoveryCodesRemaining < LOW_RECOVERY_CODES;
    return (
      <>
        <Panel
          id="mfa-title"
          title="Double authentification"
          description="Chaque connexion demande un code à 6 chiffres, en plus de votre mot de passe."
        >
          <p className="mb-4 flex flex-wrap items-center gap-3">
            <Badge tone="success">Activée</Badge>
            <span className="text-sm">
              {recoveryCodesRemaining} code{recoveryCodesRemaining > 1 ? "s" : ""} de secours
              restant{recoveryCodesRemaining > 1 ? "s" : ""}
            </span>
          </p>
          {low && (
            <p
              role="status"
              className="rounded-xl border border-rating/40 bg-rating/10 px-4 py-3 text-sm"
            >
              Il vous reste peu de codes de secours : générez-en de nouveaux ci-dessous.
            </p>
          )}
          <p className="mt-4 text-sm text-muted">
            Si vous perdez votre téléphone sans code de secours, un autre responsable de la
            plateforme doit réinitialiser votre double authentification (commande{" "}
            <code>make resetmfa EMAIL=…</code>). L&apos;interface ne le permet pas : un compte
            piraté ne doit pas pouvoir retirer sa propre protection.
          </p>
        </Panel>
        <Panel
          id="recovery-title"
          title="Codes de secours"
          description="À utiliser si vous n'avez plus accès à votre application."
        >
          <RegenerateRecoveryCodesForm />
        </Panel>
      </>
    );
  }

  return (
    <Panel
      id="mfa-title"
      title="Double authentification"
      description="Pas encore activée. Elle sera exigée de tous les modérateurs avant toute mise en ligne : activez-la dès maintenant pour protéger votre compte."
    >
      <p className="mb-4">
        <Badge tone="warning">Désactivée</Badge>
      </p>
      <MfaSetup continueHref={routes.adminSecurity} continueLabel="Terminé" />
    </Panel>
  );
}
