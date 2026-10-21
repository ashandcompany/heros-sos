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
