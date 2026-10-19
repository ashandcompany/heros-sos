"use client";

import type { ComponentPropsWithoutRef } from "react";

interface DialogTriggerProps extends Omit<ComponentPropsWithoutRef<"button">, "onClick" | "type"> {
  /** `id` de l'élément <dialog> à ouvrir. */
  dialogId: string;
}

/**
 * Ouvre un <dialog> natif rendu côté serveur. Le navigateur gère le focus piégé,
 * la touche Échap et le retour du focus sur ce bouton à la fermeture.
 */
export function DialogTrigger({ dialogId, ...props }: DialogTriggerProps) {
  return (
    <button
      type="button"
      aria-haspopup="dialog"
      onClick={() => {
        const dialog = document.getElementById(dialogId);
        if (dialog instanceof HTMLDialogElement) dialog.showModal();
      }}
      {...props}
    />
  );
}
