import type { Metadata } from "next";
import { Suspense } from "react";
import { AddHeroVideo } from "@/components/admin/AddHeroVideo";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";
import { VideosModeration } from "@/components/admin/VideosModeration";

export const metadata: Metadata = { title: "Vidéothèque" };

export default function AdminVideosPage({ searchParams }: PageProps<"/admin/videotheque">) {
  return (
    <>
      <AdminPageHeader
        title="Vidéothèque"
        description="Films ajoutés par les super-héros dans leur vidéothèque. Seuls les films publiés sont visibles des citoyens : refusez les contenus hors sujet, publicitaires ou contraires aux règles."
      />
      <Suspense fallback={<div aria-hidden="true" className="mb-8 h-16" />}>
        <AddHeroVideo searchParams={searchParams} />
      </Suspense>
      <Suspense fallback={<AdminListSkeleton />}>
        <VideosModeration searchParams={searchParams} />
      </Suspense>
    </>
  );
}
