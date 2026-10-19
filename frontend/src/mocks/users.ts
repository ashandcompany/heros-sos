import type { AuthUser } from "@/types";

interface MockAccount {
  user: AuthUser;
  email: string;
  /** Mot de passe en clair : uniquement pour les mocks, jamais côté API réelle. */
  password: string;
}

export const accountsMock: MockAccount[] = [
  {
    user: { id: "user-1", displayName: "Camille", role: "citoyen" },
    email: "citoyen@heros-sos.fr",
    password: "Citoyen123!",
  },
  {
    user: { id: "hero-1", displayName: "Lumen", role: "super-heros" },
    email: "lumen@heros-sos.fr",
    password: "Heros123!",
  },
  {
    user: { id: "admin-1", displayName: "Équipe Heros-SOS", role: "admin" },
    email: "admin@heros-sos.fr",
    password: "Admin123!",
  },
];
