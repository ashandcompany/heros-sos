import { sanitizeArticleHtml } from "@/lib/sanitize";
import { cn } from "./cn";

/** Mêmes styles que dans l'éditeur du back-office (src/components/admin/RichTextEditor.tsx). */
const contentClasses = [
  "max-w-prose leading-relaxed",
  "[&_a]:text-accent [&_a]:underline [&_blockquote]:my-6 [&_blockquote]:border-l-2 [&_blockquote]:border-accent [&_blockquote]:pl-4 [&_blockquote]:italic",
  "[&_h2]:mt-10 [&_h2]:font-display [&_h2]:text-2xl [&_h2]:font-bold [&_h3]:mt-8 [&_h3]:font-display [&_h3]:text-xl [&_h3]:font-semibold",
  "[&_hr]:my-8 [&_hr]:border-border [&_li]:my-1 [&_ol]:list-decimal [&_ol]:pl-6 [&_p]:my-4 [&_ul]:list-disc [&_ul]:pl-6",
].join(" ");

/** HTML produit par l'éditeur du back-office (article, newsletter), assaini avant affichage. */
export function RichContent({ html, className }: { html: string; className?: string }) {
  return (
    <div
      className={cn(contentClasses, className)}
      // Contenu assaini (liste blanche de balises), voir src/lib/sanitize.ts.
      dangerouslySetInnerHTML={{ __html: sanitizeArticleHtml(html) }}
    />
  );
}
