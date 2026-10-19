import type { HeroApplication } from "@/types";

/** Demandes d'inscription de super-héros. Personnages et coordonnées entièrement fictifs. */
export const heroApplicationsMock: HeroApplication[] = [
  {
    id: "application-1",
    alias: "Résonance",
    power: "Reproduit et amplifie n'importe quel son",
    civility: "madame",
    firstName: "Inès",
    lastName: "Garnier",
    birthDate: "1996-05-19",
    email: "resonance@exemple.fr",
    mobilePhone: "0655667788",
    address: { street: "12 rue des Carmes", postalCode: "31000", city: "Toulouse", country: "FR" },
    motivation:
      "Ancienne ingénieure du son, je peux neutraliser un vilain en saturant ses tympans à distance et localiser des victimes grâce à l'écholocalisation. Je souhaite rejoindre le réseau pour couvrir la région toulousaine.",
    submittedAt: "2026-10-17T16:05:00+02:00",
    status: "en-attente",
    reviewedAt: null,
  },
  {
    id: "application-2",
    alias: "Captain Mega Ultra",
    power: "Tous les pouvoirs à la fois",
    civility: "monsieur",
    firstName: "Jordan",
    lastName: "Petit",
    birthDate: "2008-02-11",
    email: "captain.mega@exemple.fr",
    mobilePhone: "0766778899",
    address: { street: "4 place du Marché", postalCode: "59000", city: "Lille", country: "FR" },
    motivation: "Je suis le plus fort de tous, je veux juste une cape officielle.",
    submittedAt: "2026-10-18T22:47:00+02:00",
    status: "en-attente",
    reviewedAt: null,
  },
];
