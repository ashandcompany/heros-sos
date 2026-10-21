"use client";

import { useActionState } from "react";
import { logoutEverywhereAction } from "@/app/actions/auth";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { idleFormState } from "@/lib/validation/form-state";

/** Ferme toutes les sessions du compte, celle-ci comprise, puis renvoie vers la connexion. */
export function LogoutEverywhereForm() {
  const [state, formAction] = useActionState(logoutEverywhereAction, idleFormState);

  return (
    <form action={formAction} className="space-y-4">
      <p className="text-sm text-pretty text-muted">
        Un appareil perdu, un ordinateur partagé ? Fermez toutes les sessions ouvertes avec votre
        compte, y compris celle-ci : il faudra vous reconnecter partout.
      </p>
      <FormMessage state={state} />
      <SubmitButton variant="danger" pendingLabel="Déconnexion…">
        Me déconnecter de tous les appareils
      </SubmitButton>
    </form>
  );
}
