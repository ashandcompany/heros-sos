/*
 * La file des actions faites hors ligne. Chaque action y est rangée (corps chiffré) avec l'identifiant qui la rend
 * rejouable sans risque, puis envoyée une à une, dans l'ordre, dès que le réseau revient. Le serveur reste
 * l'autorité : il accepte l'action, ou la refuse en expliquant pourquoi (un autre héros est déjà sur place…).
 *
 * Le module ne touche ni au réseau ni au stockage : tout passe par des paramètres (`store`, `sealer`, `send`), ce
 * qui permet de le tester sans navigateur (`npm test`).
 */

import type { CommandDraft } from "./commands.ts";
import type { Sealer } from "./seal.ts";
import type { CommandState, OutboxStore, StoredCommand } from "./store.ts";

/** Ce que l'interface affiche d'une commande (jamais son contenu). */
export interface OutboxEntry {
  id: string;
  label: string;
  createdAt: string;
  state: CommandState;
  error?: string;
}

/** Une commande déchiffrée, prête à partir. */
export interface SendableCommand {
  /** Clé d'idempotence : la même commande rejouée n'est appliquée qu'une fois. */
  id: string;
  path: string;
  body?: object;
  /** Date de création sur l'appareil (ISO 8601). */
  createdAt: string;
}

/** Ce que le serveur a répondu à une commande. */
export type SendOutcome =
  | { kind: "done" }
  /** Refusée : elle ne sera pas rejouée telle quelle, l'utilisateur en est informé. */
  | { kind: "rejected"; message: string }
  /** Pas de réponse exploitable (réseau, serveur indisponible) : on réessaiera plus tard. */
  | { kind: "retry" }
  /** Session expirée : rien ne part avant une nouvelle connexion. */
  | { kind: "unauthenticated" };

export interface FlushReport {
  synced: number;
  rejected: number;
  /** Ce qui a interrompu l'envoi, s'il reste des commandes à envoyer. */
  stoppedBy?: "retry" | "unauthenticated";
}

export interface Outbox {
  enqueue(userId: string, draft: CommandDraft): Promise<OutboxEntry>;
  /** Commandes du compte : en attente, ou refusées par le serveur. */
  list(userId: string): Promise<OutboxEntry[]>;
  /** Oublie une commande (refusée, que l'utilisateur a lue). */
  dismiss(id: string): Promise<void>;
  /** Envoie les commandes en attente du compte, une à une, dans l'ordre de leur création. */
  flush(
    userId: string,
    send: (command: SendableCommand) => Promise<SendOutcome>,
  ): Promise<FlushReport>;
}

export interface OutboxDeps {
  store: OutboxStore;
  sealer: Sealer;
  now?: () => Date;
  newId?: () => string;
}

const entryOf = ({ id, label, createdAt, state, error }: StoredCommand): OutboxEntry => ({
  id,
  label,
  createdAt,
  state,
  ...(error ? { error } : {}),
});

/** Ordre de création sur l'appareil, puis ordre d'arrivée dans la file à égalité. */
const byCreation = (first: StoredCommand, second: StoredCommand) =>
  first.createdAt === second.createdAt ? 0 : first.createdAt < second.createdAt ? -1 : 1;

export function createOutbox({
  store,
  sealer,
  now = () => new Date(),
  newId = () => crypto.randomUUID(),
}: OutboxDeps): Outbox {
  // Un seul envoi à la fois dans cet onglet : deux envois simultanés rejoueraient les mêmes commandes.
  let flushing: Promise<FlushReport> | undefined;

  async function run(
    userId: string,
    send: (command: SendableCommand) => Promise<SendOutcome>,
  ): Promise<FlushReport> {
    const report: FlushReport = { synced: 0, rejected: 0 };
    const queue = (await store.all())
      .filter((command) => command.userId === userId && command.state === "pending")
      .sort(byCreation);

    for (const command of queue) {
      const body = command.sealedBody
        ? ((await sealer.open(command.sealedBody)) as object)
        : undefined;
      const outcome = await send({
        id: command.id,
        path: command.path,
        createdAt: command.createdAt,
        ...(body ? { body } : {}),
      });

      if (outcome.kind === "done") {
        await store.remove(command.id);
        report.synced++;
        continue;
      }
      if (outcome.kind === "rejected") {
        await store.patch(command.id, { state: "rejected", error: outcome.message });
        report.rejected++;
        continue;
      }
      // Réseau ou session : on s'arrête là, et l'ordre est conservé pour la prochaine fois.
      if (outcome.kind === "retry")
        await store.patch(command.id, { attempts: command.attempts + 1 });
      report.stoppedBy = outcome.kind;
      break;
    }
    return report;
  }

  return {
    async enqueue(userId, draft) {
      const command: StoredCommand = {
        id: newId(),
        userId,
        createdAt: now().toISOString(),
        label: draft.label,
        path: draft.path,
        sealedBody: draft.body === undefined ? null : await sealer.seal(draft.body),
        state: "pending",
        attempts: 0,
      };
      await store.add(command);
      return entryOf(command);
    },

    async list(userId) {
      return (await store.all())
        .filter((command) => command.userId === userId)
        .sort(byCreation)
        .map(entryOf);
    },

    dismiss: (id) => store.remove(id),

    flush(userId, send) {
      flushing ??= run(userId, send).finally(() => {
        flushing = undefined;
      });
      return flushing;
    },
  };
}
