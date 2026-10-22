import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";
import { ImdbSearch, MyVideos } from "@/components/hero/HeroVideosSections";

export const metadata: Metadata = { title: "Ma vidéothèque" };

export default function HeroVideosPage({ searchParams }: PageProps<"/heros/films">) {
  return (
    <>
      <AdminPageHeader
        title="Ma vidéothèque"
        description="Les films qui vous sont consacrés, issus du catalogue IMDb. Chaque ajout est vérifié par la modération avant d'être visible des citoyens."
      />

      <div className="grid gap-10 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)] lg:items-start">
        <section aria-labelledby="my-videos-title">
          <h2 id="my-videos-title" className="mb-4 scroll-mt-20 font-display text-xl font-semibold">
            Mes films
          </h2>
          <Suspense fallback={<AdminListSkeleton items={2} />}>
            <MyVideos searchParams={searchParams} />
          </Suspense>
        </section>

        <section
          aria-labelledby="imdb-title"
          className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6"
        >
          <h2 id="imdb-title" className="mb-4 font-display text-xl font-semibold">
            Ajouter un film depuis IMDb
          </h2>
          <Suspense fallback={<div aria-hidden="true" className="h-11" />}>
            <ImdbSearch searchParams={searchParams} />
          </Suspense>
        </section>
      </div>
    </>
  );
}
