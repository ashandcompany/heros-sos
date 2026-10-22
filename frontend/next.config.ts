import type { NextConfig } from "next";

const apiUrl = (process.env.API_URL ?? "").replace(/\/+$/, "");

const nextConfig: NextConfig = {
  output: "standalone",
  // Photos de profil (`avatarUrl` : `/api/v1/avatars/…`, même origine que l'API). Derrière le proxy,
  // Caddy envoie `/api/v1/*` à l'API et ce rewrite ne sert jamais ; il prend le relais quand le
  // front est ouvert directement (`npm run dev` sur le port 3000).
  async rewrites() {
    return apiUrl
      ? [{ source: "/api/v1/avatars/:name", destination: `${apiUrl}/avatars/:name` }]
      : [];
  },
  // Le service worker (public/sw.js) ne doit jamais être servi depuis un cache : une version corrigée doit
  // atteindre les navigateurs tout de suite, c'est lui qui contrôle ce qu'ils gardent hors ligne.
  async headers() {
    return [
      {
        source: "/sw.js",
        headers: [
          { key: "Content-Type", value: "application/javascript; charset=utf-8" },
          { key: "Cache-Control", value: "no-cache, no-store, must-revalidate" },
        ],
      },
    ];
  },
  cacheComponents: true,
  partialPrefetching: true,
  experimental: {
    serverActions: {
      // Photo de profil : 2 Mo maximum (voir AVATAR_MAX_BYTES) + marge pour l'enveloppe multipart.
      bodySizeLimit: "3mb",
    },
  },
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
