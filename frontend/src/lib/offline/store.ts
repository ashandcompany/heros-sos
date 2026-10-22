/*
 * Stockage de la file hors ligne : IndexedDB dans le navigateur, la mémoire dans les tests. L'interface est
 * volontairement minimale : tout le comportement (ordre, reprise, refus) est dans `outbox.ts`, testé sans navigateur.
 */

import type { KeyStore, Sealed } from "./seal.ts";

export type CommandState = "pending" | "rejected";

/** Une commande telle que rangée : son corps est chiffré. */
export interface StoredCommand {
  id: string;
  /** Compte à l'origine de la commande : elle ne part que sous ce compte. */
  userId: string;
  /** Date de création sur l'appareil (ISO 8601) : elle donne l'ordre de rejeu, et le délai de synchronisation. */
  createdAt: string;
  label: string;
  path: string;
  sealedBody: Sealed | null;
  state: CommandState;
  /** Tentatives d'envoi qui ont échoué faute de réseau. */
  attempts: number;
  /** Raison du refus, pour une commande `rejected`. */
  error?: string;
}

export interface OutboxStore {
  add(command: StoredCommand): Promise<void>;
  all(): Promise<StoredCommand[]>;
  patch(
    id: string,
    changes: Partial<Pick<StoredCommand, "state" | "attempts" | "error">>,
  ): Promise<void>;
  remove(id: string): Promise<void>;
}

export function createMemoryStores(): { commands: OutboxStore; keys: KeyStore } {
  const commands = new Map<string, StoredCommand>();
  let key: CryptoKey | undefined;
  return {
    commands: {
      add: async (command) => void commands.set(command.id, command),
      all: async () => [...commands.values()],
      async patch(id, changes) {
        const current = commands.get(id);
        if (current) commands.set(id, { ...current, ...changes });
      },
      remove: async (id) => void commands.delete(id),
    },
    keys: {
      load: async () => key,
      save: async (created) => void (key = created),
    },
  };
}

const DATABASE = "heros-sos-offline";
const COMMANDS = "commands";
const KEYS = "keys";
const KEY_ID = "outbox";

function request<T>(operation: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    operation.onsuccess = () => resolve(operation.result);
    operation.onerror = () => reject(operation.error);
  });
}

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const opening = indexedDB.open(DATABASE, 1);
    opening.onupgradeneeded = () => {
      opening.result.createObjectStore(COMMANDS, { keyPath: "id" });
      opening.result.createObjectStore(KEYS);
    };
    opening.onsuccess = () => resolve(opening.result);
    opening.onerror = () => reject(opening.error);
  });
}

/** IndexedDB : la base s'ouvre à la première utilisation, une seule fois. */
export function createIndexedDbStores(): { commands: OutboxStore; keys: KeyStore } {
  let database: Promise<IDBDatabase> | undefined;
  const store = async (name: string, mode: IDBTransactionMode) => {
    database ??= openDatabase();
    return (await database).transaction(name, mode).objectStore(name);
  };

  return {
    commands: {
      async add(command) {
        await request((await store(COMMANDS, "readwrite")).add(command));
      },
      async all() {
        return request((await store(COMMANDS, "readonly")).getAll() as IDBRequest<StoredCommand[]>);
      },
      async patch(id, changes) {
        const objects = await store(COMMANDS, "readwrite");
        const current = (await request(objects.get(id))) as StoredCommand | undefined;
        if (current) await request(objects.put({ ...current, ...changes }));
      },
      async remove(id) {
        await request((await store(COMMANDS, "readwrite")).delete(id));
      },
    },
    keys: {
      async load() {
        return (await request((await store(KEYS, "readonly")).get(KEY_ID))) as
          CryptoKey | undefined;
      },
      async save(key) {
        await request((await store(KEYS, "readwrite")).put(key, KEY_ID));
      },
    },
  };
}
