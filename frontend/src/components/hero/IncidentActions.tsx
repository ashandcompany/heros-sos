"use client";

import { useActionState } from "react";
import { useFormStatus } from "react-dom";
import {
  finishInterventionAction,
  inviteHeroesAction,
  joinInterventionAction,
  respondToInvitationAction,
  setPriorityAction,
  takeChargeAction,
  voteOnOutcomeAction,
} from "@/app/actions/hero";
import { useOfflineAction } from "@/components/offline/useOfflineAction";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { SelectField } from "@/components/ui/SelectField";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { incidentPriorityLabels } from "@/lib/format";
import { heroForms } from "@/lib/offline/forms";
import { idleFormState } from "@/lib/validation/form-state";
import { OUTCOME_NOTES_MAX_LENGTH } from "@/lib/validation/hero";
import type { IncidentPriority, InterventionRef, SuperHeroSummary } from "@/types";

/* Formulaires de la page de détail d'un incident : ils fonctionnent aussi sans JavaScript. */

export function PriorityForm({
  incidentId,
  priority,
}: {
  incidentId: string;
  priority: IncidentPriority;
}) {
  const [state, formAction] = useActionState(setPriorityAction, idleFormState);

  return (
    <form action={formAction} className="space-y-3">
      <input type="hidden" name="id" value={incidentId} />
      <SelectField label="Priorité" name="priority" defaultValue={priority}>
        {Object.entries(incidentPriorityLabels).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </SelectField>
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Enregistrement…">
        Définir la priorité
      </SubmitButton>
    </form>
  );
}

export function TakeChargeForm({ incidentId }: { incidentId: string }) {
  const [state, formAction] = useActionState(
    useOfflineAction(takeChargeAction, heroForms.takeCharge),
    idleFormState,
  );

  return (
    <form action={formAction} className="space-y-3">
      <input type="hidden" name="id" value={incidentId} />
      <SubmitButton pendingLabel="Prise en charge…" className="w-full">
        Prendre en charge
      </SubmitButton>
      <FormMessage state={state} />
    </form>
  );
}

/** Champs cachés qui désignent l'intervention (`InterventionRef`). */
export function InterventionRefInputs({ interventionRef }: { interventionRef: InterventionRef }) {
  return (
    <>
      <input type="hidden" name="id" value={interventionRef.incidentId} />
      <input type="hidden" name="interventionId" value={interventionRef.interventionId} />
    </>
  );
}

interface Choice {
  value: string;
  label: string;
  variant: "primary" | "secondary" | "danger";
}

/** Un bouton d'envoi par choix : la valeur cliquée est transmise dans le champ `name`. */
export function ChoiceButtons({ name, choices }: { name: string; choices: Choice[] }) {
  const { pending } = useFormStatus();

  return (
    <div className="flex flex-wrap gap-2">
      {choices.map((choice) => (
        <button
          key={choice.value}
          type="submit"
          name={name}
          value={choice.value}
          disabled={pending}
          className={cn(buttonClasses(choice.variant), "flex-1 disabled:opacity-60")}
        >
          {choice.label}
        </button>
      ))}
    </div>
  );
}

const outcomeChoices: Choice[] = [
  { value: "reussi", label: "Réussi", variant: "primary" },
  { value: "echoue", label: "Échoué", variant: "danger" },
];

/**
 * Résultat de l'intervention. Seul, le héros la clôt ; en équipe, le résultat est soumis au vote
 * des membres confirmés.
 */
export function FinishInterventionForm({
  interventionRef,
  solo,
}: {
  interventionRef: InterventionRef;
  solo: boolean;
}) {
  const [state, formAction] = useActionState(
    useOfflineAction(finishInterventionAction, heroForms.finish),
    idleFormState,
  );

  return (
    <form action={formAction} className="space-y-3">
      <InterventionRefInputs interventionRef={interventionRef} />
      <TextAreaField
        label="Compte rendu (facultatif)"
        hint={
          solo
            ? "Joint au résultat de l'intervention."
            : "Présenté à l'équipe avec le résultat proposé."
        }
        name="notes"
        rows={3}
        maxLength={OUTCOME_NOTES_MAX_LENGTH}
      />
      <fieldset>
        <legend className="mb-2 text-sm font-medium">
          {solo ? "Résultat de l'intervention" : "Résultat à proposer à l'équipe"}
        </legend>
        <ChoiceButtons name="outcome" choices={outcomeChoices} />
      </fieldset>
      <FormMessage state={state} />
    </form>
  );
}

export function JoinTeamForm({ interventionRef }: { interventionRef: InterventionRef }) {
  const [state, formAction] = useActionState(
    useOfflineAction(joinInterventionAction, heroForms.join),
    idleFormState,
  );

  return (
    <form action={formAction} className="space-y-3">
      <InterventionRefInputs interventionRef={interventionRef} />
      <SubmitButton variant="secondary" pendingLabel="Envoi…" className="w-full">
        Rejoindre l&apos;équipe
      </SubmitButton>
      <FormMessage state={state} />
    </form>
  );
}

export function InvitationResponseForm({ interventionRef }: { interventionRef: InterventionRef }) {
  const [state, formAction] = useActionState(
    useOfflineAction(respondToInvitationAction, heroForms.respondToInvitation),
    idleFormState,
  );

  return (
    <form action={formAction} className="space-y-3">
      <InterventionRefInputs interventionRef={interventionRef} />
      <ChoiceButtons
        name="answer"
        choices={[
          { value: "accept", label: "Je viens", variant: "primary" },
          { value: "decline", label: "Je ne peux pas", variant: "secondary" },
        ]}
      />
      <FormMessage state={state} />
    </form>
  );
}

export function OutcomeVoteForm({ interventionRef }: { interventionRef: InterventionRef }) {
  const [state, formAction] = useActionState(
    useOfflineAction(voteOnOutcomeAction, heroForms.voteOnOutcome),
    idleFormState,
  );

  return (
    <form action={formAction} className="space-y-3">
      <InterventionRefInputs interventionRef={interventionRef} />
      <fieldset>
        <legend className="mb-2 text-sm font-medium">Votre vote</legend>
        <ChoiceButtons
          name="vote"
          choices={[
            { value: "approve", label: "Je valide", variant: "primary" },
            { value: "reject", label: "Je conteste", variant: "danger" },
          ]}
        />
      </fieldset>
      <FormMessage state={state} />
    </form>
  );
}

/** Invitation de super-héros : chacun reçoit un mail et doit confirmer sa présence. */
export function InviteHeroesForm({
  interventionRef,
  candidates,
}: {
  interventionRef: InterventionRef;
  /** Super-héros qui ne font pas encore partie de l'équipe, par pseudo. */
  candidates: SuperHeroSummary[];
}) {
  const [state, formAction] = useActionState(
    useOfflineAction(inviteHeroesAction, heroForms.invite),
    idleFormState,
  );

  if (candidates.length === 0) {
    return <p className="text-sm text-muted">Aucun autre super-héros à inviter.</p>;
  }

  return (
    <form action={formAction} className="space-y-3">
      <InterventionRefInputs interventionRef={interventionRef} />
      <fieldset>
        <legend className="mb-2 text-sm font-medium">Super-héros à inviter</legend>
        <div className="max-h-56 space-y-2 overflow-y-auto">
          {candidates.map((hero) => (
            <label key={hero.id} className="flex min-h-11 items-center gap-3 text-sm">
              <input
                type="checkbox"
                name="heroIds"
                value={hero.id}
                className="size-5 accent-primary"
              />
              {hero.alias}
            </label>
          ))}
        </div>
      </fieldset>
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Envoi…">
        Inviter
      </SubmitButton>
    </form>
  );
}
