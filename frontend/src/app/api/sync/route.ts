import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { isApiEnabled } from "@/lib/api/config";
import { ApiError, apiFetch } from "@/lib/api/http";
import { SESSION_COOKIE } from "@/lib/cookies";
import { isAllowedCommandPath, isUuid } from "@/lib/offline/commands";

/*
 * Relais des actions faites hors ligne : le navigateur rejoue une commande de sa file ici (`lib/offline/sender.ts`),
 * et la route la transmet à l'API Go avec la session de l'utilisateur, son jeton CSRF et la clé d'idempotence. Le
 * navigateur ne parle toujours pas directement à l'API.
 *
 * La route n'accepte que les écritures de `allowedCommandPaths` : elle ne doit jamais devenir un relais vers
 * n'importe quelle écriture de l'API avec la session de l'utilisateur.
 */

/** Une commande est un petit JSON ; au-delà, ce n'est pas une commande. */
const MAX_BODY_CHARS = 64 * 1024;

const refuse = (status: number, code: string) => NextResponse.json({ ok: false, code }, { status });

export async function POST(request: Request) {
  if (!isApiEnabled) return refuse(503, "api_disabled");

  // Un navigateur joint l'en-tête Origin aux écritures : une autre origine que la nôtre n'a rien à faire ici
  // (le cookie de session est SameSite=Lax, ceci est une seconde barrière).
  const origin = request.headers.get("origin");
  if (origin && new URL(origin).host !== request.headers.get("host")) {
    return refuse(403, "origin_not_allowed");
  }

  const raw = await request.text();
  if (raw.length > MAX_BODY_CHARS) return refuse(413, "payload_too_large");
  const command = parseCommand(raw);
  if (!command) return refuse(400, "invalid_command");

  const sessionToken = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!sessionToken) return refuse(401, "unauthenticated");

  try {
    const response = await apiFetch(command.path, {
      method: "POST",
      sessionToken,
      body: command.body,
      redirectOnUnauthorized: false,
      headers: {
        "Idempotency-Key": command.commandId,
        "X-Client-Created-At": command.createdAt,
      },
    });
    return NextResponse.json({
      ok: true,
      replayed: response.headers.get("Idempotent-Replayed") === "true",
    });
  } catch (error) {
    if (error instanceof ApiError) {
      return NextResponse.json(
        { ok: false, code: error.code, fields: error.fields },
        { status: error.status },
      );
    }
    return refuse(502, "api_unreachable");
  }
}

interface ParsedCommand {
  commandId: string;
  path: string;
  body: object | undefined;
  createdAt: string;
}

function parseCommand(raw: string): ParsedCommand | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }
  if (typeof parsed !== "object" || parsed === null) return null;
  const { commandId, path, body, createdAt } = parsed as Record<string, unknown>;

  if (typeof commandId !== "string" || !isUuid(commandId)) return null;
  if (typeof path !== "string" || !isAllowedCommandPath(path)) return null;
  if (typeof createdAt !== "string" || Number.isNaN(Date.parse(createdAt))) return null;
  if (body !== undefined && (typeof body !== "object" || body === null || Array.isArray(body))) {
    return null;
  }
  return { commandId, path, body: body as object | undefined, createdAt };
}
