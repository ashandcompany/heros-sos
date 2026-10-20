import type { HeroApplication } from "@/types";

/** Demande telle que la conserve l'API : la vue de la modération, plus le motif d'un refus. */
export interface HeroApplicationRecord extends HeroApplication {
  rejectionReason: string | null;
}

/**
 * Demandes de statut super-héros (mock en mémoire). Personnages et coordonnées entièrement
 * fictifs. Le citoyen `user-1` n'en a pas : il peut en déposer une depuis « Mon compte ».
 */
export const heroApplicationsMock: HeroApplicationRecord[] = [
  {
    userId: "applicant-1",
    alias: "Résonance",
    bio: "Reproduit et amplifie n'importe quel son.",
    requestMessage:
      "Ancienne ingénieure du son, je peux neutraliser un vilain en saturant ses tympans à distance et localiser des victimes grâce à l'écholocalisation. Je souhaite rejoindre le réseau pour couvrir la région toulousaine.",
    firstName: "Inès",
    lastName: "Garnier",
    email: "resonance@exemple.fr",
    submittedAt: "2026-10-17T16:05:00+02:00",
    status: "en-attente",
    rejectionReason: null,
  },
  {
    userId: "applicant-2",
    alias: "Captain Mega Ultra",
    bio: "Tous les pouvoirs à la fois.",
    requestMessage: "Je suis le plus fort de tous, je veux juste une cape officielle.",
    firstName: "Jordan",
    lastName: "Petit",
    email: "captain.mega@exemple.fr",
    submittedAt: "2026-10-18T22:47:00+02:00",
    status: "en-attente",
    rejectionReason: null,
  },
];
