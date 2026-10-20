import { useId, type ComponentPropsWithoutRef } from "react";
import { cn } from "./cn";

interface SelectFieldProps extends ComponentPropsWithoutRef<"select"> {
  label: string;
  error?: string;
}

/** Liste déroulante native avec label et message d'erreur reliés par `aria-describedby`. */
export function SelectField({
  label,
  error,
  id,
  className,
  children,
  ...selectProps
}: SelectFieldProps) {
  const generatedId = useId();
  const selectId = id ?? generatedId;
  const errorId = `${selectId}-error`;

  return (
    <div className={className}>
      <label htmlFor={selectId} className="mb-2 block text-sm font-medium">
        {label}
      </label>
      <select
        id={selectId}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errorId : undefined}
        className={cn(
          "min-h-12 w-full border-2 bg-background px-4 text-foreground",
          error ? "border-danger" : "border-border",
        )}
        {...selectProps}
      >
        {children}
      </select>
      {error && (
        <p id={errorId} className="mt-2 text-sm text-danger">
          {error}
        </p>
      )}
    </div>
  );
}
