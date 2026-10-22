/*
 * De la saisie d'un formulaire à l'action mise en file hors ligne : même lecture et mêmes refus que les Server
 * Actions (`app/actions/hero.ts`), mais côté navigateur. Une saisie invalide rend l'état d'erreur à afficher dans
 * le formulaire, sans rien ranger.
 */

import type { FormState } from "../validation/form-state.ts";
import { OUTCOME_NOTES_MAX_LENGTH } from "../validation/hero.ts";
import { commands, type CommandDraft } from "./commands.ts";

/**
 * Ce que rend la lecture d'un formulaire : l'action à mettre en file (et le message affiché une fois rangée), ou
 * l'état d'erreur à montrer dans le formulaire.
 */
export type BuiltCommand<State> =
  { ok: true; draft: CommandDraft; message: string } | { ok: false; state: State };

const invalid = (): BuiltCommand<FormState> => ({
  ok: false,
  state: { status: "error", message: "Requête invalide.", fieldErrors: {} },
});

const stored = (draft: CommandDraft | null, message: string): BuiltCommand<FormState> =>
  draft ? { ok: true, draft, message } : invalid();

const field = (formData: FormData, name: string) => String(formData.get(name) ?? "");

/** Formulaires des actions d'un héros sur un incident et son intervention. */
export const heroForms = {
  takeCharge: (formData: FormData) =>
    stored(
      commands.takeCharge(field(formData, "id")),
      "Prise en charge enregistrée sur cet appareil.",
    ),

  join: (formData: FormData) =>
    stored(
      commands.join(field(formData, "interventionId")),
      "Participation enregistrée sur cet appareil.",
    ),

  invite(formData: FormData): BuiltCommand<FormState> {
    const heroIds = [...new Set(formData.getAll("heroIds").map(String).filter(Boolean))];
    if (heroIds.length === 0) {
      return {
        ok: false,
        state: { status: "error", message: "Choisissez au moins un super-héros.", fieldErrors: {} },
      };
    }
    return stored(
      commands.invite(field(formData, "interventionId"), heroIds),
      "Invitation enregistrée sur cet appareil.",
    );
  },

  respondToInvitation(formData: FormData): BuiltCommand<FormState> {
    const answer = field(formData, "answer");
    if (answer !== "accept" && answer !== "decline") return invalid();
    return stored(
      commands.respondToInvitation(field(formData, "interventionId"), answer === "accept"),
      "Réponse enregistrée sur cet appareil.",
    );
  },

  finish(formData: FormData): BuiltCommand<FormState> {
    const outcome = field(formData, "outcome");
    const notes = field(formData, "notes").trim();
    if ((outcome !== "reussi" && outcome !== "echoue") || notes.length > OUTCOME_NOTES_MAX_LENGTH) {
      return invalid();
    }
    return stored(
      commands.proposeOutcome(
        field(formData, "interventionId"),
        outcome === "reussi" ? "succeeded" : "failed",
        notes,
      ),
      "Résultat enregistré sur cet appareil.",
    );
  },

  voteOnOutcome(formData: FormData): BuiltCommand<FormState> {
    const vote = field(formData, "vote");
    if (vote !== "approve" && vote !== "reject") return invalid();
    return stored(
      commands.voteOnOutcome(field(formData, "interventionId"), vote === "approve"),
      "Vote enregistré sur cet appareil.",
    );
  },
};
