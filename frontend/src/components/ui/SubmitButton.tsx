"use client";

import { useFormStatus } from "react-dom";
import { buttonClasses } from "./ButtonLink";
import { cn } from "./cn";

interface SubmitButtonProps {
  children: string;
  pendingLabel: string;
  variant?: "primary" | "secondary";
  className?: string;
}

/** Bouton d'envoi désactivé pendant l'exécution de l'action du formulaire parent. */
export function SubmitButton({
  children,
  pendingLabel,
  variant = "primary",
  className,
}: SubmitButtonProps) {
  const { pending } = useFormStatus();

  return (
    <button
      type="submit"
      disabled={pending}
      className={cn(buttonClasses(variant), "disabled:opacity-60", className)}
    >
      {pending ? pendingLabel : children}
    </button>
  );
}
