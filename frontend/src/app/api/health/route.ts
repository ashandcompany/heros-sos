/**
 * Sonde de santé du conteneur (healthcheck Docker) : répond sans rendre de page ni appeler l'API Go.
 * Interroger `/` à la place fait rendre la page d'accueil, donc deux appels à l'API, toutes les 10 secondes.
 */
export function GET() {
  return new Response("ok", { headers: { "Cache-Control": "no-store" } });
}
