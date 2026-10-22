/*
 * Les actions qu'un appareil peut mettre en file hors ligne, puis rejouer au retour du réseau (voir
 * `backend/stack/performance-et-synchronisation.md`). Une commande est une requête d'écriture de l'API, avec un
 * nom lisible pour l'utilisateur. Seules celles de cette liste sont acceptées par `/api/sync` : la route ne doit
 * pas devenir un relais vers n'importe quelle écriture de l'API avec la session de l'utilisateur.
 *
 * Aucun import à l'exécution : le module reste du TypeScript pur, testable sans navigateur ni bundler.
 */

/** Une action prête à être envoyée à l'API. */
export interface CommandDraft {
  /** Chemin de l'API, sans préfixe (`/incidents`). */
  path: string;
  /** Corps JSON (absent pour une action sans contenu). */
  body?: object;
  /** Nom lisible, affiché à l'utilisateur (« Déclaration d'incident »). */
  label: string;
}

const UUID = "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}";
const uuidPattern = new RegExp(`^${UUID}$`);

/** Écritures rejouables : la déclaration d'un incident et les actions d'une intervention. */
const allowedCommandPaths = [
  /^\/incidents$/,
  new RegExp(`^/incidents/${UUID}/interventions$`),
  new RegExp(
    `^/interventions/${UUID}/(join|invitations|invitations/me/confirm|invitations/me/decline|outcome|outcome/validate)$`,
  ),
];

export function isAllowedCommandPath(path: string): boolean {
  return allowedCommandPaths.some((pattern) => pattern.test(path));
}

export function isUuid(value: string): boolean {
  return uuidPattern.test(value);
}

/** Corps de la déclaration d'un incident (schéma `IncidentPayload` de l'API). */
export interface IncidentDeclarationBody {
  /** Identifiant choisi par l'appareil : il reconnaît son incident une fois synchronisé. */
  id: string;
  typeId: number;
  title: string;
  description: string;
  citizensRescued: number;
  /** Sans coordonnées hors ligne : le géocodage demande le réseau. */
  location: { address: string; city: string; latitude: null; longitude: null };
  occurredAt: string | null;
  villainIds: string[];
  /** « Témoin » ou « victime », tel que l'API le nomme ; `null` si le formulaire ne l'a pas demandé. */
  reporterRole?: "witness" | "victim" | null;
}

/**
 * Les constructeurs rendent `null` pour un identifiant qui n'est pas un UUID : la même requête invalide que
 * refuserait l'Server Action, sans attendre le retour du réseau pour l'apprendre.
 */
export const commands = {
  declareIncident(body: IncidentDeclarationBody): CommandDraft | null {
    return isUuid(body.id) ? { path: "/incidents", body, label: "Déclaration d'incident" } : null;
  },

  takeCharge(incidentId: string): CommandDraft | null {
    return isUuid(incidentId)
      ? { path: `/incidents/${incidentId}/interventions`, label: "Prise en charge d'un incident" }
      : null;
  },

  join(interventionId: string): CommandDraft | null {
    return isUuid(interventionId)
      ? { path: `/interventions/${interventionId}/join`, label: "Participation à l'équipe" }
      : null;
  },

  invite(interventionId: string, heroIds: string[]): CommandDraft | null {
    if (!isUuid(interventionId) || heroIds.length === 0 || !heroIds.every(isUuid)) return null;
    return {
      path: `/interventions/${interventionId}/invitations`,
      body: { heroIds },
      label: "Invitation de super-héros",
    };
  },

  respondToInvitation(interventionId: string, accept: boolean): CommandDraft | null {
    if (!isUuid(interventionId)) return null;
    return {
      path: `/interventions/${interventionId}/invitations/me/${accept ? "confirm" : "decline"}`,
      label: accept ? "Acceptation d'une invitation" : "Refus d'une invitation",
    };
  },

  proposeOutcome(
    interventionId: string,
    status: "succeeded" | "failed",
    notes: string,
  ): CommandDraft | null {
    if (!isUuid(interventionId)) return null;
    return {
      path: `/interventions/${interventionId}/outcome`,
      body: notes ? { status, notes } : { status },
      label: "Résultat d'une intervention",
    };
  },

  voteOnOutcome(interventionId: string, approved: boolean): CommandDraft | null {
    if (!isUuid(interventionId)) return null;
    return {
      path: `/interventions/${interventionId}/outcome/validate`,
      body: { approved },
      label: "Vote sur un résultat",
    };
  },
};
