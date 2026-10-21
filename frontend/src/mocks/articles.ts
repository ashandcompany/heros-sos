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
  {
    id: "article-3",
    title: "Docteur Rouille : reconnaître ses attaques",
    slug: "docteur-rouille-reconnaitre-ses-attaques",
    excerpt:
      "Bruits métalliques, odeur de fer, objets qui s'effritent : les signes qui doivent vous alerter.",
    content:
      "<p>Ingénieur déchu, <strong>Docteur Rouille</strong> corrode le métal en quelques secondes. Ses attaques visent surtout les bijouteries, les usines et les ponts.</p><h2>Les signes avant-coureurs</h2><ul><li><p>Des grincements inhabituels autour de structures métalliques.</p></li><li><p>Une poussière orangée qui se dépose au sol.</p></li><li><p>Une forte odeur de fer.</p></li></ul><h2>Que faire ?</h2><p>Éloignez-vous de toute structure métallique, puis <a href=\"https://heros-sos.fr/profil/incidents\">déclarez l'incident</a> en précisant le lieu exact.</p><blockquote><p>Ne tentez jamais de l'arrêter vous-même.</p></blockquote>",
    status: "publie",
    authorName: "Équipe Heros-SOS",
    createdAt: "2026-10-05T09:00:00+02:00",
    updatedAt: "2026-10-05T09:30:00+02:00",
    publishedAt: "2026-10-05T09:30:00+02:00",
  },
  {
    id: "article-4",
    title: "Bilan de septembre : 312 interventions réussies",
    slug: "bilan-de-septembre-312-interventions-reussies",
    excerpt: "Retour sur un mois chargé pour les super-héros du réseau, ville par ville.",
    content:
      "<p>En septembre, les super-héros du réseau ont mené <strong>312 interventions réussies</strong>, soit 18 % de plus qu'en août.</p><h2>Les villes les plus actives</h2><ol><li><p>Lyon, avec 74 interventions.</p></li><li><p>Paris, avec 69 interventions.</p></li><li><p>Marseille, avec 41 interventions.</p></li></ol><p>Merci à tous les citoyens qui déclarent les incidents avec précision : c'est grâce à vous que les héros arrivent si vite.</p>",
    status: "publie",
    authorName: "Équipe Heros-SOS",
    createdAt: "2026-10-01T08:00:00+02:00",
    updatedAt: "2026-10-01T08:00:00+02:00",
    publishedAt: "2026-10-01T08:00:00+02:00",
  },
];
