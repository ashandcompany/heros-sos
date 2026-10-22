/*
 * Chiffrement des commandes en file : adresse et description d'un incident ne restent pas en clair dans le stockage
 * du navigateur (AES-GCM 256). La clé est non exportable et vit dans le stockage du navigateur : cela protège
 * contre la lecture directe des fichiers, pas contre du code qui tournerait sur ce site (voir la doc).
 *
 * Aucun import à l'exécution : le module sert aussi aux tests (`npm test`, sans bundler).
 */

/** Contenu chiffré : le vecteur d'initialisation (propre à chaque chiffrement) et le texte chiffré. */
export interface Sealed {
  iv: ArrayBuffer;
  data: ArrayBuffer;
}

/** Où la clé est conservée entre deux visites (IndexedDB en production, la mémoire dans les tests). */
export interface KeyStore {
  load(): Promise<CryptoKey | undefined>;
  save(key: CryptoKey): Promise<void>;
}

export interface Sealer {
  seal(value: unknown): Promise<Sealed>;
  open(sealed: Sealed): Promise<unknown>;
}

const IV_BYTES = 12;

export function createSealer(keys: KeyStore, subtle: SubtleCrypto = crypto.subtle): Sealer {
  // Une seule clé, créée à la première utilisation : deux appels simultanés ne doivent pas en fabriquer deux.
  let pending: Promise<CryptoKey> | undefined;
  const key = () => {
    pending ??= (async () => {
      const existing = await keys.load();
      if (existing) return existing;
      const created = await subtle.generateKey({ name: "AES-GCM", length: 256 }, false, [
        "encrypt",
        "decrypt",
      ]);
      await keys.save(created);
      return created;
    })();
    return pending;
  };

  return {
    async seal(value) {
      const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES));
      const plain = new TextEncoder().encode(JSON.stringify(value));
      const data = await subtle.encrypt({ name: "AES-GCM", iv }, await key(), plain);
      return { iv: iv.buffer, data };
    },

    async open(sealed) {
      const plain = await subtle.decrypt(
        { name: "AES-GCM", iv: sealed.iv },
        await key(),
        sealed.data,
      );
      return JSON.parse(new TextDecoder().decode(plain));
    },
  };
}
