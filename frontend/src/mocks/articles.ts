import type { BlogArticle } from "@/types";

export const articlesMock: BlogArticle[] = [
  {
    id: "article-1",
    title: "Comment bien déclarer un incident",
    slug: "comment-bien-declarer-un-incident",
    excerpt:
      "Lieu précis, description, vilain impliqué : les informations qui font gagner du temps.",
    content:
      "<p>Une déclaration précise permet au super-héros le plus proche d'intervenir <strong>plus vite</strong>.</p><h2>Les informations essentielles</h2><ul><li><p>Le lieu exact, avec un repère visible.</p></li><li><p>Le nombre de personnes en danger.</p></li><li><p>Le vilain impliqué, si vous le reconnaissez.</p></li></ul>",
    status: "publie",
    authorName: "Équipe Heros-SOS",
    createdAt: "2026-09-02T10:00:00+02:00",
    updatedAt: "2026-09-02T10:00:00+02:00",
    publishedAt: "2026-09-02T10:00:00+02:00",
  },
  {
    id: "article-2",
    title: "Bienvenue aux nouveaux héros de l'automne",
    slug: "bienvenue-aux-nouveaux-heros-de-l-automne",
    excerpt: "Présentation des super-héros qui ont rejoint le réseau ce trimestre.",
    content: "<p>Brouillon : présenter chaque nouveau héros en deux phrases.</p>",
    status: "brouillon",
    authorName: "Équipe Heros-SOS",
    createdAt: "2026-10-15T14:20:00+02:00",
    updatedAt: "2026-10-18T09:05:00+02:00",
    publishedAt: null,
  },
];
