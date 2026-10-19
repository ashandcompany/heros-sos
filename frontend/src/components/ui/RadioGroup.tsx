import { useId } from "react";
import { cn } from "./cn";

interface RadioGroupProps {
  legend: string;
  name: string;
  options: Array<{ value: string; label: string }>;
  defaultValue?: string;
  error?: string;
  required?: boolean;
}

/** Groupe de boutons radio dans un fieldset, avec message d'erreur accessible. */
export function RadioGroup({
  legend,
  name,
  options,
  defaultValue,
  error,
  required,
}: RadioGroupProps) {
  const id = useId();
  const errorId = `${id}-error`;

  return (
    <fieldset aria-describedby={error ? errorId : undefined}>
      <legend className="mb-2 text-sm font-medium">{legend}</legend>
      <div className="flex flex-wrap gap-2">
        {options.map((option) => (
          <label
            key={option.value}
            className={cn(
              "flex min-h-11 cursor-pointer items-center gap-2 rounded-xl border bg-background px-4 text-sm has-checked:border-primary has-checked:bg-primary/10 has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-primary",
              error ? "border-danger" : "border-border",
            )}
          >
            <input
              type="radio"
              name={name}
              value={option.value}
              defaultChecked={option.value === defaultValue}
              required={required}
              className="size-4 accent-primary focus-visible:outline-none"
            />
            {option.label}
          </label>
        ))}
      </div>
      {error && (
        <p id={errorId} className="mt-2 text-sm text-danger">
          {error}
        </p>
      )}
    </fieldset>
  );
}
