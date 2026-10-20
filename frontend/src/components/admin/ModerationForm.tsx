"use client";

import { useActionState } from "react";
import { useFormStatus } from "react-dom";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { idleFormState, type FormState } from "@/lib/validation/form-state";

export interface Decision {
  /** Valeur transmise dans `decision` : un statut de modération ou une action (`deactivate`…). */
  value: string;
  label: string;
  variant?: "primary" | "secondary" | "danger";
}

type ModerationAction = (previousState: FormState, formData: FormData) => Promise<FormState>;

function DecisionButtons({ decisions, subject }: { decisions: Decision[]; subject: string }) {
  const { pending } = useFormStatus();

  return (
    <div className="flex flex-wrap gap-2">
      {decisions.map((decision) => (
        <button
          key={decision.value}
          type="submit"
          name="decision"
          value={decision.value}
          disabled={pending}
          className={cn(buttonClasses(decision.variant ?? "secondary"), "disabled:opacity-60")}
        >
          {decision.label}
          <span className="sr-only"> : {subject}</span>
        </button>
      ))}
    </div>
  );
}

/**
 * Boutons de décision d'un élément à modérer (un bouton d'envoi par décision, la valeur
 * cliquée est transmise dans `decision`). Fonctionne sans JavaScript.
 */
export function ModerationForm({
  action,
  id,
  decisions,
  subject,
  className,
}: {
  action: ModerationAction;
  /** Identifiant de l'élément modéré, transmis dans `id`. */
  id: string;
  decisions: Decision[];
  /** Nom de l'élément, ajouté au libellé des boutons pour les lecteurs d'écran. */
  subject: string;
  className?: string;
}) {
  const [state, formAction] = useActionState(action, idleFormState);

  return (
    <form action={formAction} className={cn("space-y-3", className)}>
      <input type="hidden" name="id" value={id} />
      <DecisionButtons decisions={decisions} subject={subject} />
      <FormMessage state={state} />
    </form>
  );
}
