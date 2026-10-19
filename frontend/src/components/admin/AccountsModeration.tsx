import { moderateAccountAction, reviewEmailChangeAction } from "@/app/actions/admin";
import { Avatar } from "@/components/ui/Avatar";
import { getCitizenAccounts } from "@/lib/api";
import { formatDateTime, formatFullName, formatPhone } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import type { CitizenAccount, ModerationStatus } from "@/types";
import { decisionsFor } from "./decisions";
import { ModerationBadge } from "./ModerationBadge";
import {
  EmptyState,
  ModerationFilter,
  readModerationFilter,
  type ModerationSearchParams,
} from "./ModerationFilter";
import { ModerationForm } from "./ModerationForm";

const accountStatusLabels: Record<ModerationStatus, string> = {
  "en-attente": "À contrôler",
  valide: "Actif",
  rejete: "Suspendu",
};

function AccountCard({ account }: { account: CitizenAccount }) {
  const fullName = formatFullName(account.lastName, account.firstName) ?? account.email;
  const titleId = `compte-${account.id}-titre`;

  return (
    <article
      aria-labelledby={titleId}
      className="grid gap-5 border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center"
    >
      <div className="flex min-w-0 items-start gap-4">
        <Avatar name={account.firstName} src={account.avatarUrl} size={48} />
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 id={titleId} className="font-display text-lg font-semibold">
              {fullName}
            </h2>
            <ModerationBadge status={account.moderation} labels={accountStatusLabels} />
          </div>
          <p className="mt-1 text-sm break-all">
            <a href={`mailto:${account.email}`} className="text-accent hover:underline">
              {account.email}
            </a>
            <span className="text-muted"> · {formatPhone(account.mobilePhone)}</span>
          </p>
          <p className="mt-1 text-sm text-muted">
            {account.address.city} · Inscrit le{" "}
            <time dateTime={account.registeredAt} className="font-mono">
              {formatDateTime(account.registeredAt)}
            </time>
          </p>
        </div>
      </div>
      <ModerationForm
        action={moderateAccountAction}
        id={account.id}
        subject={fullName}
        decisions={decisionsFor(account.moderation, {
          approve: "Valider le compte",
          reject: "Suspendre",
          restore: "Réactiver",
        })}
      />
    </article>
  );
}

/** Comptes citoyens à contrôler. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function AccountsModeration({
  searchParams,
}: {
  searchParams: ModerationSearchParams;
}) {
  const token = await requireAdminSession();
  const filter = await readModerationFilter(searchParams);
  const accounts = await getCitizenAccounts(token);
  const listed = accounts.filter((account) => !filter || account.moderation === filter);
  const emailChanges = accounts.filter((account) => account.pendingEmail);

  return (
    <div className="space-y-12">
      <section aria-labelledby="emails" className="scroll-mt-32">
        <h2 id="emails" className="mb-1 font-display text-xl font-semibold">
          Changements d&apos;adresse email
        </h2>
        <p className="mb-4 text-sm text-muted">
          La nouvelle adresse ne remplace l&apos;ancienne qu&apos;après votre validation.
        </p>
        {emailChanges.length === 0 ? (
          <EmptyState>Aucune demande en attente.</EmptyState>
        ) : (
          <ul className="space-y-3">
            {emailChanges.map((account) => {
              const fullName = formatFullName(account.lastName, account.firstName) ?? "";
              return (
                <li
                  key={account.id}
                  className="grid gap-4 border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center"
                >
                  <div className="min-w-0">
                    <p className="font-display font-semibold">{fullName}</p>
                    <dl className="mt-2 grid gap-x-6 gap-y-1 text-sm sm:grid-cols-[auto_1fr]">
                      <dt className="text-muted">Adresse actuelle</dt>
                      <dd className="break-all">{account.email}</dd>
                      <dt className="text-muted">Nouvelle adresse</dt>
                      <dd className="font-medium break-all text-accent">{account.pendingEmail}</dd>
                    </dl>
                  </div>
                  <ModerationForm
                    action={reviewEmailChangeAction}
                    id={account.id}
                    subject={`changement d'email de ${fullName}`}
                    decisions={[
                      { value: "valide", label: "Accepter", variant: "primary" },
                      { value: "rejete", label: "Refuser", variant: "danger" },
                    ]}
                  />
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <section aria-labelledby="accounts-title">
        <h2 id="accounts-title" className="mb-1 font-display text-xl font-semibold">
          Comptes citoyens
        </h2>
        <p className="mb-4 text-sm text-muted">
          Contrôlez les comptes créés récemment. Un compte suspendu ne peut plus se connecter ni
          déclarer d&apos;incident.
        </p>
        <ModerationFilter
          basePath={routes.adminAccounts}
          current={filter}
          labels={{
            "en-attente": "À contrôler",
            valide: "Actifs",
            rejete: "Suspendus",
            tous: "Tous",
          }}
        />
        {listed.length === 0 ? (
          <EmptyState>Aucun compte dans cette catégorie.</EmptyState>
        ) : (
          <ul className="space-y-3">
            {listed.map((account) => (
              <li key={account.id}>
                <AccountCard account={account} />
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
