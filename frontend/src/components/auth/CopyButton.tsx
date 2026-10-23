"use client";

import { useState } from "react";
import { buttonClasses } from "@/components/ui/ButtonLink";

const FEEDBACK_MS = 2500;

/**
 * Copie un texte dans le presse-papiers (la clé de la 2FA, à coller dans une application d'ordinateur qui ne
 * s'ouvre pas depuis un lien). Le texte n'est jamais affiché ni envoyé ailleurs que dans le presse-papiers.
 */
export function CopyButton({
  text,
  label,
  copiedLabel,
}: {
  text: string;
  label: string;
  copiedLabel: string;
}) {
  const [state, setState] = useState<"idle" | "copied" | "failed">("idle");

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setState("copied");
    } catch {
      // Presse-papiers refusé (permission, contexte non sécurisé) : la clé reste sélectionnable à côté.
      setState("failed");
    }
    setTimeout(() => setState("idle"), FEEDBACK_MS);
  }

  return (
    <>
      <button type="button" onClick={copy} className={buttonClasses("secondary", "sm")}>
        {state === "copied" ? copiedLabel : label}
      </button>
      <span role="status" className="sr-only">
        {state === "copied"
          ? copiedLabel
          : state === "failed"
            ? "Copie impossible : sélectionnez la clé."
            : ""}
      </span>
    </>
  );
}
