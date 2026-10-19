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
        description="Commentaires laissés par les citoyens en notant un super-héros. Masquez les propos injurieux ou les données personnelles : la note, elle, reste prise en compte."
      />
      <Suspense fallback={<AdminListSkeleton />}>
        <CommentsModeration searchParams={searchParams} />
      </Suspense>
    </>
  );
}
