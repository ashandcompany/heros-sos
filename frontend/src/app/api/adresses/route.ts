import { NextResponse, type NextRequest } from "next/server";
import { searchAddresses, type AddressSearchKind } from "@/lib/geocoding";

const MIN_QUERY_LENGTH = 3;
const MAX_QUERY_LENGTH = 120;

/**
 * Suggestions d'adresses pour l'autocomplétion des formulaires.
 * `GET /api/adresses?q=14 rue de la république&type=postal|place`
 * Réponse : `{ suggestions: AddressSuggestion[] }` (liste vide si la saisie est trop courte).
 */
export async function GET(request: NextRequest) {
  const params = request.nextUrl.searchParams;
  const query = (params.get("q") ?? "").trim().replace(/\s+/g, " ");
  const kind: AddressSearchKind = params.get("type") === "place" ? "place" : "postal";

  if (query.length < MIN_QUERY_LENGTH || query.length > MAX_QUERY_LENGTH) {
    return NextResponse.json({ suggestions: [] });
  }

  try {
    const suggestions = await searchAddresses(query, kind);
    return NextResponse.json(
      { suggestions },
      { headers: { "Cache-Control": "private, max-age=600" } },
    );
  } catch (error) {
    console.error("Autocomplétion d'adresse indisponible :", error);
    // Le formulaire reste utilisable : l'utilisateur saisit simplement son adresse à la main.
    return NextResponse.json({ suggestions: [] }, { status: 502 });
  }
}
