import { villainsMock } from "@/mocks/villains";
import type { Villain } from "@/types";
import { mockResponse } from "./config";

/**
 * Tous les super-vilains connus, triés par nom.
 * Future implémentation : `GET ${API_URL}/villains`
 */
export async function getVillains(): Promise<Villain[]> {
  return mockResponse([...villainsMock].sort((a, b) => a.alias.localeCompare(b.alias, "fr")));
}

/** Future implémentation : `GET ${API_URL}/villains/${id}` (404 → `null`). */
export async function getVillain(id: string): Promise<Villain | null> {
  return mockResponse(villainsMock.find((villain) => villain.id === id) ?? null);
}
