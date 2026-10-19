import { useId, type ComponentPropsWithoutRef, type ReactNode } from "react";
import { cn } from "./cn";

interface TextFieldProps extends Omit<ComponentPropsWithoutRef<"input">, "children"> {
  label: string;
  hint?: string;
  error?: string;
  /** Contenu affiché à droite du label (ex : lien « Mot de passe oublié ? »). */
  labelAside?: ReactNode;
  /** Contenu placé dans le champ, à droite (ex : bouton afficher/masquer). */
  trailing?: ReactNode;
}

/** Champ texte avec label, aide et message d'erreur reliés par `aria-describedby`. */
export function TextField({
  label,
  hint,
  error,
  labelAside,
  trailing,
  id,
  className,
  ...inputProps
}: TextFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const hintId = `${inputId}-hint`;
  const errorId = `${inputId}-error`;
  const describedBy = [hint && hintId, error && errorId].filter(Boolean).join(" ") || undefined;

  return (
    <div className={className}>
      <div className="mb-2 flex items-baseline justify-between gap-4">
        <label htmlFor={inputId} className="text-sm font-medium">
          {label}
        </label>
        {labelAside}
      </div>
      <div className="relative">
        <input
          id={inputId}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          className={cn(
            "min-h-12 w-full rounded-xl border bg-background px-4 text-foreground placeholder:text-muted",
            error ? "border-danger" : "border-border",
            trailing ? "pr-14" : undefined,
          )}
          {...inputProps}
        />
        {trailing && <div className="absolute inset-y-0 right-1 flex items-center">{trailing}</div>}
      </div>
      {hint && (
        <p id={hintId} className="mt-2 text-sm text-muted">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} className="mt-2 text-sm text-danger">
          {error}
        </p>
      )}
    </div>
  );
}
