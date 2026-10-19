"use client";

import { useActionState } from "react";
import { useFormStatus } from "react-dom";
import { addVideoAction } from "@/app/actions/hero";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { idleFormState } from "@/lib/validation/form-state";

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

/** Ajoute un film trouvé sur IMDb à la vidéothèque (publié après modération). */
export function AddVideoForm({ imdbId, title }: { imdbId: string; title: string }) {
  const [state, formAction] = useActionState(addVideoAction, idleFormState);

  return (
    <form action={formAction} className="space-y-2">
      <input type="hidden" name="imdbId" value={imdbId} />
      <AddButton title={title} />
      <FormMessage state={state} />
    </form>
  );
}
