import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { CommentsModeration } from "@/components/admin/CommentsModeration";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";

export const metadata: Metadata = { title: "Commentaires" };

export default function AdminCommentsPage({ searchParams }: PageProps<"/admin/commentaires">) {
  return (
    <>
      <AdminPageHeader
        title="Commentaires"
        description="Avis laissés par les citoyens sur les super-héros, et réponses des super-héros à ces avis."
      />
      <Suspense fallback={<AdminListSkeleton />}>
        <CommentsModeration searchParams={searchParams} />
      </Suspense>
    </>
  );
}
