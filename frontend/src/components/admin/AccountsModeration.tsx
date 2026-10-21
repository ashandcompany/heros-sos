import Link from "next/link";
import { moderateAccountAction, reviewEmailChangeAction } from "@/app/actions/admin";
import { Avatar } from "@/components/ui/Avatar";
import { Badge } from "@/components/ui/Badge";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { getAccounts, getEmailChangeRequests } from "@/lib/api";
import {
  accountStatusLabels,
  formatDate,
  formatDateTime,
  formatFullName,
  formatInteger,
  isAccountStatus,
  userRoleLabels,
} from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import type { AccountStatus, AccountSummary, EmailChangeRequest } from "@/types";
import { EmptyState } from "./ModerationFilter";
import { ModerationForm, type Decision } from "./ModerationForm";
import { Pagination } from "./Pagination";

const PAGE_SIZE = 20;

/** Paramètres d'URL de la liste : `?statut=…&email=…&page=…`. */
export type AccountsSearchParams = Promise<{
  statut?: string | string[];
  email?: string | string[];
  page?: string | string[];
}>;

const first = (value: string | string[] | undefined) => (Array.isArray(value) ? value[0] : value);

const statusTones = {
  "en-attente-activation": "warning",
  actif: "success",
  desactive: "danger",
} as const;

/** Actions possibles selon l'état du compte. Un compte de modérateur n'en a aucune. */
function decisionsFor(account: AccountSummary): Decision[] {
  if (account.role === "admin") return [];
  if (account.status === "desactive") {
    return [{ value: "reactivate", label: "Réactiver", variant: "secondary" }];
  }
  const decisions: Decision[] = [];
  if (account.status === "actif" && account.role === "citoyen" && !account.verifiedAt) {
    decisions.push({ value: "verify", label: "Marquer comme vérifié", variant: "primary" });
  }
  decisions.push({ value: "deactivate", label: "Désactiver", variant: "danger" });
  return decisions;
}

function AccountCard({ account }: { account: AccountSummary }) {
  const fullName = formatFullName(account.lastName, account.firstName) ?? account.email;
  const displayName = account.alias ? `${account.alias} (${fullName})` : fullName;
  const titleId = `compte-${account.id}-titre`;
  const decisions = decisionsFor(account);

  return (
    <article
      aria-labelledby={titleId}
      className="grid gap-5 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center"
    >
      <div className="flex min-w-0 items-start gap-4">
        <Avatar name={account.alias ?? account.firstName} src={null} size={48} />
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 id={titleId} className="font-display text-lg font-semibold">
              {displayName}
            </h2>
            <Badge tone={statusTones[account.status]}>{accountStatusLabels[account.status]}</Badge>
            {account.role !== "citoyen" && <Badge>{userRoleLabels[account.role]}</Badge>}
            {account.verifiedAt && (
              <Badge tone="accent">
                Vérifié<span className="sr-only"> le {formatDate(account.verifiedAt)}</span>
              </Badge>
            )}
          </div>
          <p className="mt-1 text-sm break-all">
            <a href={`mailto:${account.email}`} className="text-accent hover:underline">
              {account.email}
            </a>
          </p>
          <p className="mt-1 text-sm text-muted">
            Inscrit le{" "}
            <time dateTime={account.registeredAt} className="font-mono">
              {formatDateTime(account.registeredAt)}
            </time>
          </p>
        </div>
      </div>
      {decisions.length > 0 && (
        <ModerationForm
          action={moderateAccountAction}
          id={account.id}
          subject={displayName}
          decisions={decisions}
        />
      )}
    </article>
  );
}

const filterOptions: Array<{ value: AccountStatus | undefined; label: string }> = [
  { value: undefined, label: "Tous" },
  { value: "actif", label: "Actifs" },
  { value: "en-attente-activation", label: "Non activés" },
  { value: "desactive", label: "Désactivés" },
];

/** Onglets d'état et recherche par email (liens et formulaire GET : fonctionnent sans JavaScript). */
function AccountsFilters({ status, email }: { status?: AccountStatus; email?: string }) {
  const href = (value: AccountStatus | undefined) => {
    const params = new URLSearchParams();
    if (value) params.set("statut", value);
    if (email) params.set("email", email);
    const search = params.toString();
    return search ? `${routes.adminAccounts}?${search}` : routes.adminAccounts;
  };

  return (
    <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
      <nav aria-label="Filtrer par état">
        <ul className="flex flex-wrap gap-2">
          {filterOptions.map((option) => {
            const isCurrent = status === option.value;
            return (
              <li key={option.label}>
                <Link
                  href={href(option.value)}
                  aria-current={isCurrent ? "page" : undefined}
                  className={cn(
                    "inline-flex min-h-10 items-center rounded-full border px-4 text-sm font-medium",
                    isCurrent
                      ? "border-primary bg-primary/15 text-foreground"
                      : "border-border text-muted hover:border-muted hover:text-foreground",
                  )}
                >
                  {option.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
      <form role="search" action={routes.adminAccounts} className="flex flex-wrap items-end gap-2">
        {status && <input type="hidden" name="statut" value={status} />}
        <label className="text-sm font-medium">
          <span className="mb-2 block">Rechercher par email</span>
          <input
            type="search"
            name="email"
            defaultValue={email}
            placeholder="Début de l'adresse"
            className="min-h-10 w-64 max-w-full border-2 border-border bg-background px-3 text-foreground placeholder:text-muted"
          />
        </label>
        <button type="submit" className={buttonClasses("secondary")}>
          Rechercher
        </button>
      </form>
    </div>
  );
}

function EmailChanges({ requests }: { requests: EmailChangeRequest[] }) {
  return (
    <section aria-labelledby="emails" className="scroll-mt-32">
      <h2 id="emails" className="mb-1 font-display text-xl font-semibold">
        Changements d&apos;adresse email
      </h2>
      <p className="mb-4 text-sm text-muted">
        La nouvelle adresse ne remplace l&apos;ancienne qu&apos;après votre validation.
      </p>
      {requests.length === 0 ? (
        <EmptyState>Aucune demande en attente.</EmptyState>
      ) : (
        <ul className="space-y-3">
          {requests.map((request) => {
            const fullName = formatFullName(request.lastName, request.firstName) ?? "";
            return (
              <li
                key={request.id}
                className="grid gap-4 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center"
              >
                <div className="min-w-0">
                  <p className="font-display font-semibold">{fullName}</p>
                  <dl className="mt-2 grid gap-x-6 gap-y-1 text-sm sm:grid-cols-[auto_1fr]">
                    <dt className="text-muted">Adresse actuelle</dt>
                    <dd className="break-all">{request.currentEmail}</dd>
                    <dt className="text-muted">Nouvelle adresse</dt>
                    <dd className="font-medium break-all text-accent">{request.newEmail}</dd>
                  </dl>
                </div>
                <ModerationForm
                  action={reviewEmailChangeAction}
                  id={request.id}
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
  );
}

/** Comptes de la plateforme. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function AccountsModeration({ searchParams }: { searchParams: AccountsSearchParams }) {
  const token = await requireAdminSession();
  const params = await searchParams;
  const rawStatus = first(params.statut);
  const status = rawStatus && isAccountStatus(rawStatus) ? rawStatus : undefined;
  const email = first(params.email)?.trim() || undefined;
  const pageNumber = Math.max(1, Math.trunc(Number(first(params.page))) || 1);

  const [page, emailChanges] = await Promise.all([
    getAccounts(token, { status, email, limit: PAGE_SIZE, offset: (pageNumber - 1) * PAGE_SIZE }),
    getEmailChangeRequests(token),
  ]);

  return (
    <div className="space-y-12">
      <EmailChanges requests={emailChanges} />

      <section aria-labelledby="accounts-title">
        <h2 id="accounts-title" className="mb-1 font-display text-xl font-semibold">
          Comptes
        </h2>
        <p className="mb-4 text-sm text-muted">
          Un compte désactivé est aussitôt déconnecté et ne peut plus se connecter. Marquer un
          citoyen comme vérifié est une information de confiance : elle ne donne aucun droit
          supplémentaire.
        </p>
        <AccountsFilters status={status} email={email} />
        {page.items.length === 0 ? (
          <EmptyState>Aucun compte ne correspond à ces critères.</EmptyState>
        ) : (
          <>
            <p className="mb-3 text-sm text-muted">
              {formatInteger(page.total)} compte{page.total > 1 ? "s" : ""}
            </p>
            <ul className="space-y-3">
              {page.items.map((account) => (
                <li key={account.id}>
                  <AccountCard account={account} />
                </li>
              ))}
            </ul>
            <Pagination
              page={page}
              basePath={routes.adminAccounts}
              query={{ statut: status, email }}
            />
          </>
        )}
      </section>
    </div>
  );
}
