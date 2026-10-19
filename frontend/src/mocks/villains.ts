import type { Villain } from "@/types";

export const villainsMock: Villain[] = [
  {
    id: "villain-1",
    alias: "Le Corbeau Gris",
    threatLevel: "eleve",
    description: "Cambrioleur insaisissable qui commande des nuées d'oiseaux mécaniques.",
    reportedIncidentsCount: 12,
    status: "en-liberte",
  },
  {
    id: "villain-2",
    alias: "Docteur Rouille",
    threatLevel: "modere",
    description: "Ingénieur déchu capable de corroder n'importe quel métal en quelques secondes.",
    reportedIncidentsCount: 8,
    status: "en-liberte",
  },
  {
    id: "villain-3",
    alias: "Néréide Noire",
    threatLevel: "critique",
    description: "Contrôle les eaux portuaires et s'en prend aux musées et aux navires.",
    reportedIncidentsCount: 5,
    status: "en-liberte",
  },
  {
    id: "villain-4",
    alias: "Le Faussaire",
    threatLevel: "faible",
    description: "Maître de l'illusion, spécialiste des faux billets et des fausses identités.",
    reportedIncidentsCount: 17,
    status: "emprisonne",
  },
  {
    id: "villain-5",
    alias: "Volt Sauvage",
    threatLevel: "eleve",
    description: "Absorbe et relâche l'électricité, cible privilégiée : les réseaux urbains.",
    reportedIncidentsCount: 9,
    status: "en-liberte",
  },
  {
    id: "villain-6",
    alias: "Madame Sablier",
    threatLevel: "critique",
    description: "Ralentit le temps autour d'elle ; impliquée dans plusieurs prises d'otages.",
    reportedIncidentsCount: 4,
    status: "emprisonne",
  },
];
