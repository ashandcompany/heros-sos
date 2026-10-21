"use client";

import { useActionState } from "react";
import { useFormStatus } from "react-dom";
import { addVideoAction } from "@/app/actions/hero";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { idleFormState, type FormState } from "@/lib/validation/form-state";

function AddButton({ title }: { title: string }) {
  const { pending } = useFormStatus();

  return (
    <button
      type="submit"
      disabled={pending}
      className={cn(buttonClasses("secondary"), "disabled:opacity-60")}
    >
      {pending ? "Ajout…" : "Ajouter"}
      <span className="sr-only"> : {title}</span>
    </button>
  );
}

/**
 * Ajoute un film trouvé sur IMDb à une vidéothèque : celle du héros connecté (publié après
 * modération, par défaut), ou celle de `heroId` avec l'action du back-office (publié aussitôt).
 */
export function AddVideoForm({
  imdbId,
  title,
  heroId,
  action = addVideoAction,
}: {
  imdbId: string;
  title: string;
  heroId?: string;
  action?: (previousState: FormState, formData: FormData) => Promise<FormState>;
}) {
  const [state, formAction] = useActionState(action, idleFormState);

  return (
    <form action={formAction} className="space-y-2">
      <input type="hidden" name="imdbId" value={imdbId} />
      {heroId && <input type="hidden" name="heroId" value={heroId} />}
      <AddButton title={title} />
      <FormMessage state={state} />
    </form>
  );
}
