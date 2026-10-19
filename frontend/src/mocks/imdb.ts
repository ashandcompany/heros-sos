import type { ImdbTitle } from "@/types";

/** Extrait fictif du catalogue IMDb (identifiants inventés), utilisé tant que l'API n'existe pas. */
export const imdbCatalogMock: ImdbTitle[] = [
  {
    id: "tt9900101",
    title: "Lumen : La Nuit des Corbeaux",
    year: 2025,
    plot: "Lyon plonge dans le noir et une héroïne de lumière traque une nuée d'oiseaux mécaniques.",
  },
  {
    id: "tt9900102",
    title: "Lumen 2 : Éclipse",
    year: 2026,
    plot: "Face à un vilain capable d'absorber la lumière, Lumen doit apprendre à se battre dans l'ombre.",
  },
  {
    id: "tt9900201",
    title: "Rempart",
    year: 2024,
    plot: "Un homme ordinaire découvre qu'il peut générer des boucliers d'énergie indestructibles.",
  },
  {
    id: "tt9900301",
    title: "Vélocité : Mur du son",
    year: 2023,
    plot: "Le héros le plus rapide de France affronte un saboteur du réseau électrique.",
  },
  {
    id: "tt9900401",
    title: "Les Gardiens de la Presqu'île",
    year: 2022,
    plot: "Documentaire sur les super-héros qui veillent sur le centre de Lyon.",
  },
  {
    id: "tt9900501",
    title: "Héros malgré eux",
    year: 2021,
    plot: "Une comédie sur une brigade de super-héros débutants et leurs premières interventions.",
  },
];
