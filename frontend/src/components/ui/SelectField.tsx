import { useId, type ComponentPropsWithoutRef } from "react";
import { cn } from "./cn";

interface SelectFieldProps extends ComponentPropsWithoutRef<"select"> {
  label: string;
  /** Aide affichée sous la liste. */
  hint?: string;
  error?: string;
}

/** Liste déroulante native avec label, aide et message d'erreur reliés par `aria-describedby`. */
export function SelectField({
  label,
  hint,
  error,
  id,
  className,
  children,
  ...selectProps
}: SelectFieldProps) {
  const generatedId = useId();
  const selectId = id ?? generatedId;
  const hintId = `${selectId}-hint`;
  const errorId = `${selectId}-error`;
  const describedBy = [hint && hintId, error && errorId].filter(Boolean).join(" ") || undefined;

  return (
    <div className={className}>
      <label htmlFor={selectId} className="mb-2 block text-sm font-medium">
        {label}
      </label>
      <select
        id={selectId}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={cn(
          "min-h-12 w-full border-2 bg-background px-4 text-foreground",
          error ? "border-danger" : "border-border",
        )}
        {...selectProps}
      >
        {children}
      </select>
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
