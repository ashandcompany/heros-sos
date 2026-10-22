/*
 * Service worker de Heros-SOS : lecture hors ligne et réveil de la synchronisation.
 *
 *   - /_next/static/*   : fichiers de l'application, gardés une fois chargés (cache d'abord) ;
 *   - pages /profil et /heros : gardées après chaque visite, servies si le réseau manque (réseau d'abord) ;
 *   - le reste (API, /api/sync, flux temps réel…) : jamais touché, toujours le réseau.
 *
 * L'envoi des actions faites hors ligne est dans la page (src/components/offline), pas ici : ce worker réveille
 * seulement les pages ouvertes quand le navigateur lui signale le retour du réseau (Background Sync).
 */

const VERSION = "v1";
const STATIC_CACHE = `heros-static-${VERSION}`;
// Le préfixe « heros-pages- » est aussi connu de la page, qui efface ces caches à la déconnexion.
const PAGE_CACHE = `heros-pages-${VERSION}`;
const OFFLINE_URL = "/hors-ligne";
const BACKGROUND_SYNC_TAG = "heros-outbox";
// Après ce délai sans réponse, on sert la copie gardée plutôt que d'attendre un réseau qui ne revient pas.
const NETWORK_TIMEOUT_MS = 4000;
// Pages gardées pour la lecture hors ligne : les espaces personnels (voir `protectedAreas` dans src/proxy.ts).
const CACHED_AREAS = ["/profil", "/heros"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(STATIC_CACHE)
      .then((cache) => cache.add(OFFLINE_URL))
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      // Les caches d'une version précédente ne servent plus.
      for (const name of await caches.keys()) {
        const ours = name.startsWith("heros-static-") || name.startsWith("heros-pages-");
        if (ours && name !== STATIC_CACHE && name !== PAGE_CACHE) await caches.delete(name);
      }
      await self.clients.claim();
    })(),
  );
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;
  const url = new URL(request.url);
  // L'API est servie sur la même origine par le reverse proxy : /api/ (API Go, relais de synchronisation, flux
  // temps réel) ne passe jamais par le cache.
  if (url.origin !== self.location.origin || url.pathname.startsWith("/api/")) return;

  if (url.pathname.startsWith("/_next/static/")) {
    event.respondWith(cacheFirst(request));
  } else if (request.mode === "navigate") {
    event.respondWith(navigate(request, url));
  }
});

async function cacheFirst(request) {
  const cache = await caches.open(STATIC_CACHE);
  const cached = await cache.match(request);
  if (cached) return cached;
  const response = await fetch(request);
  if (response.ok) await cache.put(request, response.clone());
  return response;
}

function withTimeout(promise, ms) {
  return Promise.race([
    promise,
    new Promise((_, reject) => setTimeout(() => reject(new Error("network timeout")), ms)),
  ]);
}

async function navigate(request, url) {
  try {
    const response = await withTimeout(fetch(request), NETWORK_TIMEOUT_MS);
    // Une redirection (session expirée → connexion) n'est pas la page demandée : on ne la garde pas.
    const keep =
      response.ok &&
      !response.redirected &&
      CACHED_AREAS.some((area) => url.pathname === area || url.pathname.startsWith(`${area}/`));
    if (keep) {
      const cache = await caches.open(PAGE_CACHE);
      await cache.put(request, response.clone());
    }
    return response;
  } catch {
    const cached = await (await caches.open(PAGE_CACHE)).match(request);
    if (cached) return cached;
    const offline = await (await caches.open(STATIC_CACHE)).match(OFFLINE_URL);
    return offline ?? Response.error();
  }
}

self.addEventListener("message", (event) => {
  if (event.data && event.data.type === "clear-pages") event.waitUntil(caches.delete(PAGE_CACHE));
});

// Le navigateur signale le retour du réseau : les pages ouvertes envoient leur file.
self.addEventListener("sync", (event) => {
  if (event.tag !== BACKGROUND_SYNC_TAG) return;
  event.waitUntil(
    self.clients
      .matchAll({ type: "window", includeUncontrolled: true })
      .then((windows) => windows.forEach((window) => window.postMessage({ type: "flush" }))),
  );
});
