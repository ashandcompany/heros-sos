import sanitizeHtml from "sanitize-html";

/**
 * Assainit le HTML d'un article avant son affichage public. L'API devra aussi le faire à
 * l'enregistrement : ce filtre est une seconde barrière. La liste blanche correspond aux
 * blocs proposés par l'éditeur du back-office (src/components/admin/RichTextEditor.tsx).
 */
export function sanitizeArticleHtml(html: string): string {
  return sanitizeHtml(html, {
    allowedTags: [
      "p",
      "br",
      "h2",
      "h3",
      "strong",
      "em",
      "s",
      "ul",
      "ol",
      "li",
      "blockquote",
      "hr",
      "a",
    ],
    // `rel` n'est jamais repris de la source : il est imposé par `transformTags` ci-dessous.
    allowedAttributes: { a: ["href", "rel"] },
    allowedSchemes: ["http", "https", "mailto"],
    // Liens externes : pas de transmission du référent ni d'accès à la page d'origine.
    transformTags: {
      a: sanitizeHtml.simpleTransform("a", { rel: "noopener noreferrer nofollow" }),
    },
  });
}
