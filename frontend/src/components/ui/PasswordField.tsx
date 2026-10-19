"use client";

import { useId, useState, type ComponentProps } from "react";
import { TextField } from "./TextField";

function EyeIcon({ crossed }: { crossed: boolean }) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      className="size-5 fill-none stroke-current"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z" />
      <circle cx="12" cy="12" r="3" />
      {crossed && <path d="M3 3l18 18" />}
    </svg>
  );
}

type PasswordFieldProps = Omit<ComponentProps<typeof TextField>, "type" | "trailing">;

/** Champ mot de passe avec bouton afficher / masquer. */
export function PasswordField({ id, ...props }: PasswordFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const [visible, setVisible] = useState(false);

  return (
    <TextField
      {...props}
      id={inputId}
      type={visible ? "text" : "password"}
      trailing={
        <button
          type="button"
          onClick={() => setVisible((value) => !value)}
          aria-controls={inputId}
          aria-pressed={visible}
          aria-label="Afficher le mot de passe"
          className="inline-flex size-11 items-center justify-center rounded-lg text-muted hover:text-foreground"
        >
          <EyeIcon crossed={visible} />
        </button>
      }
    />
  );
}
