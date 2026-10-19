import { useId, type ComponentPropsWithoutRef } from "react";
import { cn } from "./cn";

interface TextAreaFieldProps extends ComponentPropsWithoutRef<"textarea"> {
  label: string;
  hint?: string;
  error?: string;
}

/** Zone de texte avec label, aide et message d'erreur reliés par `aria-describedby`. */
export function TextAreaField({
  label,
  hint,
  error,
  id,
  className,
  rows = 4,
  ...textareaProps
}: TextAreaFieldProps) {
  const generatedId = useId();
  const textareaId = id ?? generatedId;
  const hintId = `${textareaId}-hint`;
  const errorId = `${textareaId}-error`;
  const describedBy = [hint && hintId, error && errorId].filter(Boolean).join(" ") || undefined;

  return (
    <div className={className}>
      <label htmlFor={textareaId} className="mb-2 block text-sm font-medium">
        {label}
      </label>
      <textarea
        id={textareaId}
        rows={rows}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={cn(
          "w-full rounded-xl border bg-background px-4 py-3 text-foreground placeholder:text-muted",
          error ? "border-danger" : "border-border",
        )}
        {...textareaProps}
      />
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
