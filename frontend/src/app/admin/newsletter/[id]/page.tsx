import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { deleteNewsletterAction } from "@/app/actions/admin";
import { ConfirmDeleteButton } from "@/components/admin/ConfirmDeleteButton";
import { NewsletterForm } from "@/components/admin/NewsletterForm";
import { SendNewsletterButton } from "@/components/admin/SendNewsletterButton";
import { Badge } from "@/components/ui/Badge";
import { RichContent } from "@/components/ui/RichContent";
import { getNewsletterIssue } from "@/lib/api";
import { formatDateTime, formatInteger } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import type { NewsletterIssue } from "@/types";

export const metadata: Metadata = { title: "Newsletter" };

const panelClasses =
  "max-w-3xl border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8";

/** Nombre de destinataires annoncé après un envoi (`?envoyee=…`), `null` sinon. */
async function readSentCount(
  searchParams: Promise<Record<string, string | string[] | undefined>>,
): Promise<number | null> {
  const raw = (await searchParams).envoyee;
  const value = Number(Array.isArray(raw) ? raw[0] : raw);
  return Number.isInteger(value) && value > 0 ? value : null;
}

function SentIssue({
  issue,
  sentCount,
}: {
  issue: NewsletterIssue & { sentAt: string };
  sentCount: number | null;
}) {
  return (
    <>
      {sentCount !== null && (
        <p
          role="status"
          className="mb-6 max-w-3xl border-2 border-border bg-surface-raised p-4 text-sm"
        >
          Newsletter confiée à l&apos;envoi pour {formatInteger(sentCount)} destinataire
          {sentCount > 1 ? "s" : ""}. Les mails partent en arrière-plan.
        </p>
      )}
      <div className="mb-8">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone="success">Envoyée</Badge>
          {issue.recipientUserId && <Badge tone="warning">Un seul destinataire</Badge>}
        </div>
        <h1 className="mt-3 font-display text-3xl font-bold">{issue.subject}</h1>
        <p className="mt-2 text-sm text-muted">
          Créée le <time dateTime={issue.createdAt}>{formatDateTime(issue.createdAt)}</time> ·
          Envoyée le <time dateTime={issue.sentAt}>{formatDateTime(issue.sentAt)}</time>
        </p>
      </div>
      <section aria-label="Contenu envoyé" className={panelClasses}>
        <RichContent html={issue.body} />
      </section>
    </>
  );
}

function DraftIssue({ issue }: { issue: NewsletterIssue }) {
  return (
    <>
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge tone="paper">Brouillon</Badge>
            {issue.recipientUserId && <Badge tone="warning">Un seul destinataire</Badge>}
          </div>
          <h1 className="mt-3 font-display text-3xl font-bold">Modifier la newsletter</h1>
          <p className="mt-2 text-sm text-muted">
            Créée le <time dateTime={issue.createdAt}>{formatDateTime(issue.createdAt)}</time>.
            Enregistrez vos modifications avant d&apos;envoyer.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <ConfirmDeleteButton
            action={deleteNewsletterAction}
            id={issue.id}
            subject={issue.subject}
            description="Le brouillon sera définitivement supprimé. Cette action est irréversible."
          />
          <SendNewsletterButton
            id={issue.id}
            subject={issue.subject}
            targeted={issue.recipientUserId !== null}
          />
        </div>
      </div>
      <div className={panelClasses}>
        <NewsletterForm issue={issue} />
      </div>
    </>
  );
}

/** Lit la session et les paramètres d'URL : rendu sous <Suspense>. */
async function NewsletterIssueView({ params, searchParams }: PageProps<"/admin/newsletter/[id]">) {
  const token = await requireAdminSession();
  const { id } = await params;
  const issue = await getNewsletterIssue(token, id);
  if (!issue) notFound();

  return issue.sentAt ? (
    <SentIssue
      issue={{ ...issue, sentAt: issue.sentAt }}
      sentCount={await readSentCount(searchParams)}
    />
  ) : (
    <DraftIssue issue={issue} />
  );
}

export default function AdminNewsletterPage(props: PageProps<"/admin/newsletter/[id]">) {
  return (
    <>
      <p className="mb-4 text-sm">
        <Link href={routes.adminNewsletters} className="text-accent hover:underline">
          ← Newsletters
        </Link>
      </p>
      <Suspense
        fallback={
          <div
            aria-hidden="true"
            className="h-[40rem] border-2 border-border bg-surface motion-safe:animate-pulse"
          />
        }
      >
        <NewsletterIssueView {...props} />
      </Suspense>
    </>
  );
}
