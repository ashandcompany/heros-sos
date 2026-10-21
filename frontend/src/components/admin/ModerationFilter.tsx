import Link from "next/link";
import type { ReactNode } from "react";
import { cn } from "@/components/ui/cn";
import { isModerationStatus } from "@/lib/format";
import type { ModerationStatus } from "@/types";

/** Paramètre d'URL `?statut=` des listes à modérer. */
export type ModerationSearchParams = Promise<{ statut?: string | string[] }>;

type FilterOption = ModerationStatus | "tous";

/**
 * Filtre lu dans l'URL : `fallback` par défaut (« en-attente », la file de travail des
 * modérateurs), « tous » pour tout afficher (`undefined`).
 */
export async function readModerationFilter(
  searchParams: ModerationSearchParams,
  fallback: ModerationStatus = "en-attente",
): Promise<ModerationStatus | undefined> {
  const raw = (await searchParams).statut;
  const value = Array.isArray(raw) ? raw[0] : raw;
  if (value === "tous") return undefined;
  return value && isModerationStatus(value) ? value : fallback;
}

/**
 * Onglets de filtre (liens : fonctionnent sans JavaScript et l'URL reste partageable).
 * `basePath` peut porter d'autres paramètres (`/admin/commentaires?type=reponses`).
 */
export function ModerationFilter({
  basePath,
  current,
  labels,
  options = ["en-attente", "valide", "rejete", "tous"],
  defaultOption = "en-attente",
}: {
  basePath: string;
  current: ModerationStatus | undefined;
  labels: Partial<Record<FilterOption, string>>;
  options?: FilterOption[];
  /** Option affichée sans paramètre `statut` (celle de `readModerationFilter`). */
  defaultOption?: ModerationStatus;
}) {
  const separator = basePath.includes("?") ? "&" : "?";

  return (
    <nav aria-label="Filtrer par statut" className="mb-6">
      <ul className="flex flex-wrap gap-2">
        {options.map((option) => {
          const isCurrent = (current ?? "tous") === option;
          return (
            <li key={option}>
              <Link
                href={
                  option === defaultOption ? basePath : `${basePath}${separator}statut=${option}`
                }
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
