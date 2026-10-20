import Link from "next/link";
import type { ReactNode } from "react";
import { cn } from "@/components/ui/cn";
import { isModerationStatus } from "@/lib/format";
import type { ModerationStatus } from "@/types";

/** Paramètre d'URL `?statut=` des listes à modérer. */
export type ModerationSearchParams = Promise<{ statut?: string | string[] }>;

/**
 * Filtre lu dans l'URL : « en-attente » par défaut (la file de travail des modérateurs),
 * « tous » pour tout afficher (`undefined`).
 */
export async function readModerationFilter(
  searchParams: ModerationSearchParams,
): Promise<ModerationStatus | undefined> {
  const raw = (await searchParams).statut;
  const value = Array.isArray(raw) ? raw[0] : raw;
  if (value === "tous") return undefined;
  return value && isModerationStatus(value) ? value : "en-attente";
}

/** Onglets de filtre (liens : fonctionnent sans JavaScript et l'URL reste partageable). */
export function ModerationFilter({
  basePath,
  current,
  labels,
}: {
  basePath: string;
  current: ModerationStatus | undefined;
  labels: Record<ModerationStatus | "tous", string>;
}) {
  const options: Array<ModerationStatus | "tous"> = ["en-attente", "valide", "rejete", "tous"];

  return (
    <nav aria-label="Filtrer par statut" className="mb-6">
      <ul className="flex flex-wrap gap-2">
        {options.map((option) => {
          const isCurrent = (current ?? "tous") === option;
          return (
            <li key={option}>
              <Link
                href={option === "en-attente" ? basePath : `${basePath}?statut=${option}`}
                aria-current={isCurrent ? "page" : undefined}
                className={cn(
                  "inline-flex min-h-10 items-center rounded-full border px-4 text-sm font-medium",
                  isCurrent
                    ? "border-primary bg-primary/15 text-foreground"
                    : "border-border text-muted hover:border-muted hover:text-foreground",
                )}
              >
                {labels[option]}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

export function EmptyState({ children }: { children: ReactNode }) {
  return (
    <p className="rounded-2xl border border-dashed border-border p-5 text-sm text-muted">
      {children}
    </p>
  );
}

export function AdminListSkeleton({ items = 3 }: { items?: number }) {
  return (
    <div aria-hidden="true" className="space-y-3">
      {Array.from({ length: items }, (_, index) => (
        <div
          key={index}
          className="h-40 border-2 border-border bg-surface motion-safe:animate-pulse"
        />
      ))}
    </div>
  );
}
