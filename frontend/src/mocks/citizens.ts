import type { CitizenAccount } from "@/types";

/** Comptes citoyens (mock en mémoire, modifiable par les actions de « Mon compte »). */
export const citizensMock: CitizenAccount[] = [
  {
    id: "user-1",
    civility: "madame",
    firstName: "Camille",
    lastName: "Martin",
    birthDate: "1992-04-08",
    email: "citoyen@heros-sos.fr",
    pendingEmail: null,
    mobilePhone: "0611223344",
    address: {
      street: "14 rue de la République",
      complement: "Bâtiment B, 3e étage",
      postalCode: "69002",
      city: "Lyon",
      country: "FR",
    },
    avatarUrl: null,
    newsletterSubscribed: true,
    registeredAt: "2025-06-12T09:30:00+02:00",
  },
];
