import { describe, expect, it } from "vitest";
import { commands, isAllowedCommandPath } from "./commands.ts";
import { createOutbox, type SendableCommand, type SendOutcome } from "./outbox.ts";
import { createSealer } from "./seal.ts";
import { refusalMessage, sendCommand, SYNC_ENDPOINT } from "./sender.ts";
import { createMemoryStores } from "./store.ts";

/** Comme `assert.ok` de Node : échoue si la valeur est fausse, et affine son type pour la suite du test. */
function ok(value: unknown, message?: string): asserts value {
  expect(value, message).toBeTruthy();
}

const incidentId = "0b8c1c3e-6f1d-4c7e-9a52-3d2f7b9e4a10";
const interventionId = "5c1d7e92-3a40-4b86-9f0e-6d8a2b4c1e75";

function newOutbox() {
  const stores = createMemoryStores();
  let tick = 0;
  let counter = 0;
  const outbox = createOutbox({
    store: stores.commands,
    sealer: createSealer(stores.keys),
    // Une seconde de plus à chaque commande : l'ordre de création est celui de l'appel.
    now: () => new Date(Date.UTC(2026, 9, 22, 10, 0, tick++)),
    newId: () => `command-${++counter}`,
  });
  return { outbox, stores };
}

const declare = () => commands.takeCharge(incidentId)!;
const sentLog = () => {
  const sent: SendableCommand[] = [];
  return { sent, record: (command: SendableCommand) => void sent.push(command) };
};

describe("seal", () => {
  it("chiffre et déchiffre, avec un vecteur propre à chaque chiffrement", async () => {
    const sealer = createSealer(createMemoryStores().keys);
    const value = { description: "Deux individus forcent la porte", city: "Lyon" };

    const first = await sealer.seal(value);
    const second = await sealer.seal(value);

    expect(await sealer.open(first)).toEqual(value);
    expect(new Uint8Array(first.iv)).not.toEqual(new Uint8Array(second.iv));
    ok(
      !new TextDecoder().decode(first.data).includes("Lyon"),
      "le texte ne doit pas rester en clair",
    );
  });

  it("réutilise la clé enregistrée, et n'en crée qu'une même sous des appels simultanés", async () => {
    const stores = createMemoryStores();
    let saved = 0;
    const keys = {
      ...stores.keys,
      save: async (key: CryptoKey) => (saved++, stores.keys.save(key)),
    };

    const sealer = createSealer(keys);
    const [a, b] = await Promise.all([sealer.seal("a"), sealer.seal("b")]);
    expect(saved).toBe(1);

    // Une nouvelle visite : un autre sealer retrouve la clé et lit ce qui a été chiffré avant.
    expect(await createSealer(stores.keys).open(a)).toBe("a");
    expect(await createSealer(stores.keys).open(b)).toBe("b");
  });

  it("refuse un contenu altéré", async () => {
    const sealer = createSealer(createMemoryStores().keys);
    const sealed = await sealer.seal({ ok: true });
    new Uint8Array(sealed.data)[0] ^= 0xff;
    await expect(sealer.open(sealed)).rejects.toThrow();
  });
});

describe("commands", () => {
  it("ne produit que des chemins que /api/sync accepte", () => {
    const drafts = [
      commands.takeCharge(incidentId),
      commands.join(interventionId),
      commands.invite(interventionId, [incidentId]),
      commands.respondToInvitation(interventionId, true),
      commands.respondToInvitation(interventionId, false),
      commands.proposeOutcome(interventionId, "succeeded", ""),
      commands.voteOnOutcome(interventionId, true),
      commands.declareIncident({
        id: incidentId,
        typeId: 1,
        title: "Titre",
        description: "Description",
        citizensRescued: 0,
        location: { address: "1 rue", city: "Lyon", latitude: null, longitude: null },
        occurredAt: null,
        villainIds: [],
      }),
    ];
    for (const draft of drafts) {
      ok(draft && isAllowedCommandPath(draft.path), `chemin refusé : ${draft?.path}`);
    }
  });

  it("refuse tout le reste : la route ne relaie pas n'importe quelle écriture", () => {
    for (const path of [
      "/admin/interventions/" + interventionId + "/decision",
      "/auth/logout-all",
      "/me",
      "/incidents/" + incidentId,
      "/incidents/../auth/logout",
      "/interventions/x/join",
      `/interventions/${interventionId}/join/extra`,
    ]) {
      expect(isAllowedCommandPath(path), path).toBe(false);
    }
  });

  it("refuse un identifiant qui n'est pas un UUID, sans attendre le réseau", () => {
    expect(commands.takeCharge("incident-1")).toBe(null);
    expect(commands.join("../etc")).toBe(null);
    expect(commands.invite(interventionId, [])).toBe(null);
    expect(commands.invite(interventionId, ["pas-un-uuid"])).toBe(null);
  });

  it("n'envoie les notes que si elles existent", () => {
    expect(commands.proposeOutcome(interventionId, "failed", "")?.body).toEqual({
      status: "failed",
    });
    expect(commands.proposeOutcome(interventionId, "failed", "Trop tard")?.body).toEqual({
      status: "failed",
      notes: "Trop tard",
    });
  });
});

describe("outbox", () => {
  it("range les commandes sans laisser leur contenu en clair, et les rejoue dans l'ordre de création", async () => {
    const { outbox, stores } = newOutbox();
    await outbox.enqueue("user-1", commands.takeCharge(incidentId)!);
    await outbox.enqueue("user-1", commands.voteOnOutcome(interventionId, true)!);
    await outbox.enqueue(
      "user-1",
      commands.proposeOutcome(interventionId, "succeeded", "Tout va bien")!,
    );

    const stored = await stores.commands.all();
    ok(stored.every((command) => !JSON.stringify(command).includes("Tout va bien")));

    const { sent, record } = sentLog();
    const report = await outbox.flush(
      "user-1",
      async (command) => (record(command), { kind: "done" }),
    );

    expect(report).toEqual({ synced: 3, rejected: 0 });
    expect(sent.map((command) => command.id)).toEqual(["command-1", "command-2", "command-3"]);
    expect(sent[2]?.body).toEqual({ status: "succeeded", notes: "Tout va bien" });
    expect(sent[0]?.createdAt).toBe("2026-10-22T10:00:00.000Z");
    expect(await outbox.list("user-1"), "une commande envoyée sort de la file").toEqual([]);
  });

  it("s'arrête au premier échec réseau et reprend au même endroit, sans rien perdre ni doubler", async () => {
    const { outbox } = newOutbox();
    for (const draft of [
      declare(),
      commands.join(interventionId)!,
      commands.voteOnOutcome(interventionId, true)!,
    ]) {
      await outbox.enqueue("user-1", draft);
    }

    let calls = 0;
    const flaky = async (): Promise<SendOutcome> =>
      ++calls === 2 ? { kind: "retry" } : { kind: "done" };
    const first = await outbox.flush("user-1", flaky);
    expect(first).toEqual({ synced: 1, rejected: 0, stoppedBy: "retry" });
    expect((await outbox.list("user-1")).map((entry) => entry.id)).toEqual([
      "command-2",
      "command-3",
    ]);

    const { sent, record } = sentLog();
    const second = await outbox.flush(
      "user-1",
      async (command) => (record(command), { kind: "done" }),
    );
    expect(second).toEqual({ synced: 2, rejected: 0 });
    expect(
      sent.map((command) => command.id),
      "la commande déjà envoyée n'est pas renvoyée, l'ordre est conservé",
    ).toEqual(["command-2", "command-3"]);
  });

  it("garde une commande refusée avec sa raison, et continue avec les suivantes", async () => {
    const { outbox } = newOutbox();
    await outbox.enqueue("user-1", declare());
    await outbox.enqueue("user-1", commands.join(interventionId)!);

    const report = await outbox.flush("user-1", async (command) =>
      command.id === "command-1"
        ? { kind: "rejected", message: "Un autre héros est déjà sur place." }
        : { kind: "done" },
    );

    expect(report).toEqual({ synced: 1, rejected: 1 });
    const left = await outbox.list("user-1");
    expect(left.length).toBe(1);
    expect(left[0]?.state).toBe("rejected");
    expect(left[0]?.error).toBe("Un autre héros est déjà sur place.");

    // Une commande refusée n'est jamais renvoyée toute seule ; l'utilisateur la ferme.
    const { sent, record } = sentLog();
    await outbox.flush("user-1", async (command) => (record(command), { kind: "done" }));
    expect(sent.length).toBe(0);
    await outbox.dismiss("command-1");
    expect(await outbox.list("user-1")).toEqual([]);
  });

  it("s'arrête quand la session a expiré, sans rien refuser", async () => {
    const { outbox } = newOutbox();
    await outbox.enqueue("user-1", declare());
    await outbox.enqueue("user-1", commands.join(interventionId)!);

    const report = await outbox.flush("user-1", async () => ({ kind: "unauthenticated" }));

    expect(report).toEqual({ synced: 0, rejected: 0, stoppedBy: "unauthenticated" });
    expect((await outbox.list("user-1")).filter((entry) => entry.state === "pending").length).toBe(
      2,
    );
  });

  it("n'envoie que les commandes du compte connecté", async () => {
    const { outbox } = newOutbox();
    await outbox.enqueue("alice", declare());
    await outbox.enqueue("bob", commands.join(interventionId)!);

    const { sent, record } = sentLog();
    await outbox.flush("bob", async (command) => (record(command), { kind: "done" }));

    expect(sent.map((command) => command.path)).toEqual([`/interventions/${interventionId}/join`]);
    expect(
      (await outbox.list("alice")).length,
      "la commande d'Alice attend sa prochaine connexion",
    ).toBe(1);
  });

  it("ne lance pas deux envois à la fois", async () => {
    const { outbox } = newOutbox();
    await outbox.enqueue("user-1", declare());
    let release: () => void = () => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    let calls = 0;
    const slow = async (): Promise<SendOutcome> => (calls++, await gate, { kind: "done" });

    const first = outbox.flush("user-1", slow);
    const second = outbox.flush("user-1", slow);
    release();

    expect(await first).toEqual(await second);
    expect(calls).toBe(1);
  });
});

describe("sender", () => {
  const command: SendableCommand = {
    id: "command-1",
    path: "/incidents",
    body: { title: "Titre" },
    createdAt: "2026-10-22T10:00:00.000Z",
  };
  const answer =
    (status: number, body: object = {}) =>
    async () =>
      new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
      });

  it("envoie la commande, sa clé d'idempotence et sa date à /api/sync", async () => {
    let seen: { url: string; init: RequestInit } | undefined;
    const outcome = await sendCommand(command, async (url, init) => {
      seen = { url: String(url), init: init ?? {} };
      return new Response("{}", { status: 200 });
    });

    expect(outcome).toEqual({ kind: "done" });
    expect(seen?.url).toBe(SYNC_ENDPOINT);
    expect(seen?.init.method).toBe("POST");
    expect(JSON.parse(String(seen?.init.body))).toEqual({
      commandId: "command-1",
      path: "/incidents",
      body: { title: "Titre" },
      createdAt: "2026-10-22T10:00:00.000Z",
    });
  });

  it("distingue ce qui se réessaie, ce qui demande une connexion et ce qui est refusé", async () => {
    const network = async (): Promise<Response> => {
      throw new TypeError("Failed to fetch");
    };
    expect(await sendCommand(command, network)).toEqual({ kind: "retry" });
    for (const status of [429, 500, 502, 503, 504]) {
      expect(await sendCommand(command, answer(status)), String(status)).toEqual({ kind: "retry" });
    }
    expect(await sendCommand(command, answer(401))).toEqual({ kind: "unauthenticated" });

    const refused = await sendCommand(
      command,
      answer(409, { code: "intervention_already_active" }),
    );
    expect(refused).toEqual({
      kind: "rejected",
      message: "Un autre héros est déjà sur place : rejoignez son équipe.",
    });
  });

  it("tient un incident déjà enregistré pour livré : le même envoi avait déjà abouti", async () => {
    expect(await sendCommand(command, answer(409, { code: "incident_already_exists" }))).toEqual({
      kind: "done",
    });
  });

  it("explique un refus inconnu sans inventer", () => {
    expect(refusalMessage("code_inconnu", 409)).toBe("Refusée par le serveur (code_inconnu).");
    expect(refusalMessage(undefined, 418)).toBe("Refusée par le serveur (statut 418).");
  });
});

describe("forms", () => {
  const fields = (entries: Record<string, string | string[]>) => {
    const formData = new FormData();
    for (const [name, value] of Object.entries(entries)) {
      for (const one of [value].flat()) formData.append(name, one);
    }
    return formData;
  };

  it("lit une prise en charge comme la Server Action", async () => {
    const { heroForms } = await import("./forms.ts");
    const built = heroForms.takeCharge(fields({ id: incidentId }));
    ok(built.ok);
    expect(built.draft.path).toBe(`/incidents/${incidentId}/interventions`);
    expect(built.message).toMatch(/enregistrée sur cet appareil/);
  });

  it("traduit le résultat « réussi » ou « échoué » de l'interface en statut de l'API", async () => {
    const { heroForms } = await import("./forms.ts");
    const success = heroForms.finish(
      fields({ interventionId, outcome: "reussi", notes: "  Tout va bien " }),
    );
    ok(success.ok);
    expect(success.draft.body).toEqual({ status: "succeeded", notes: "Tout va bien" });
    const failure = heroForms.finish(fields({ interventionId, outcome: "echoue" }));
    ok(failure.ok);
    expect(failure.draft.body).toEqual({ status: "failed" });
  });

  it("refuse la saisie que la Server Action refuserait, sans rien ranger", async () => {
    const { heroForms } = await import("./forms.ts");
    for (const built of [
      heroForms.takeCharge(fields({ id: "pas-un-uuid" })),
      heroForms.join(fields({})),
      heroForms.respondToInvitation(fields({ interventionId, answer: "peut-être" })),
      heroForms.voteOnOutcome(fields({ interventionId, vote: "abstention" })),
      heroForms.finish(fields({ interventionId, outcome: "reussi", notes: "x".repeat(1001) })),
      heroForms.finish(fields({ interventionId, outcome: "mitigé" })),
    ]) {
      expect(built.ok).toBe(false);
      ok(!built.ok && built.state.status === "error");
    }
  });

  it("demande au moins un héros à inviter, et ne l'invite qu'une fois", async () => {
    const { heroForms } = await import("./forms.ts");
    const none = heroForms.invite(fields({ interventionId }));
    ok(!none.ok && none.state.message === "Choisissez au moins un super-héros.");

    const twice = heroForms.invite(fields({ interventionId, heroIds: [incidentId, incidentId] }));
    ok(twice.ok);
    expect(twice.draft.body).toEqual({ heroIds: [incidentId] });
  });

  it("traduit « Je viens » et « Je ne peux pas » en confirmation et refus", async () => {
    const { heroForms } = await import("./forms.ts");
    const accept = heroForms.respondToInvitation(fields({ interventionId, answer: "accept" }));
    const decline = heroForms.respondToInvitation(fields({ interventionId, answer: "decline" }));
    ok(accept.ok && decline.ok);
    ok(accept.draft.path.endsWith("/invitations/me/confirm"));
    ok(decline.draft.path.endsWith("/invitations/me/decline"));
  });
});
