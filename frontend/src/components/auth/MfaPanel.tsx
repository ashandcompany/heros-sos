import Link from "next/link";
import { Suspense } from "react";
import { MfaSetup } from "@/components/auth/MfaSetup";
import { RegenerateRecoveryCodesForm } from "@/components/auth/RegenerateRecoveryCodesForm";
import { Panel } from "@/components/citizen/Panel";
import { Badge } from "@/components/ui/Badge";
import { getMfaOverview } from "@/lib/api";
import { routes } from "@/lib/routes";
import { requireRole } from "@/lib/server-session";
import type { UserRole } from "@/types";

/** Seuil à partir duquel il faut penser à renouveler les codes de secours. */
const LOW_RECOVERY_CODES = 3;

/** Page où ramène « Terminé » une fois la 2FA activée : celle qui affiche le panneau. */
const homeByRole: Record<UserRole, string> = {
  admin: routes.adminSecurity,
  citoyen: routes.profile,
  "super-heros": routes.heroProfile,
};

/**
 * Double authentification du compte connecté. Obligatoire pour un modérateur (`admin`), facultative pour un
 * citoyen ou un super-héros. Lit la session : à rendre sous <Suspense> (voir `MfaSection`).
 */
export async function MfaPanel({ role }: { role: UserRole }) {
  const { token } = await requireRole(role);
  const { status, recoveryCodesRemaining } = await getMfaOverview(token);
  const moderator = role === "admin";

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
            {moderator ? (
              <>
                Si vous perdez votre téléphone sans code de secours, un autre responsable de la
                plateforme doit réinitialiser votre double authentification (commande{" "}
                <code>make resetmfa EMAIL=…</code>).
              </>
            ) : (
              <>
                Si vous perdez votre téléphone sans code de secours,{" "}
                <Link href={routes.contact} className="underline">
                  contactez l&apos;équipe Heros-SOS
                </Link>{" "}
                : elle réinitialisera votre double authentification après avoir vérifié votre
                identité.
              </>
            )}{" "}
            L&apos;interface ne le permet pas : un compte piraté ne doit pas pouvoir retirer sa
            propre protection.
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
      description={
        moderator
          ? "Pas encore activée. Elle sera exigée de tous les modérateurs avant toute mise en ligne : activez-la dès maintenant pour protéger votre compte."
          : "Facultative, mais recommandée : en plus de votre mot de passe, chaque connexion demande un code à 6 chiffres généré par une application d'authentification (Google Authenticator, Aegis, 1Password…)."
      }
    >
      <p className="mb-4">
        <Badge tone="warning">Désactivée</Badge>
      </p>
      <MfaSetup continueHref={homeByRole[role]} continueLabel="Terminé" />
    </Panel>
  );
}

/** `MfaPanel` et son repli de chargement : le panneau lit la session, il doit être rendu sous <Suspense>. */
export function MfaSection({ role }: { role: UserRole }) {
  return (
    <Suspense
      fallback={
        <div
          aria-hidden="true"
          className="h-64 border-2 border-border bg-surface motion-safe:animate-pulse"
        />
      }
    >
      <MfaPanel role={role} />
    </Suspense>
  );
}
