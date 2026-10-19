import type { FormState } from "@/lib/validation/form-state";

/** Message global d'un formulaire : confirmation (status) ou erreur (alert). */
export function FormMessage({ state }: { state: FormState }) {
  if (!state.message) return null;

  if (state.status === "success") {
    return (
      <p
        role="status"
        className="rounded-xl border border-success/40 bg-success/10 px-4 py-3 text-sm text-success"
      >
        {state.message}
      </p>
    );
  }

  return (
    <p
      role="alert"
      className="rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-sm text-danger"
    >
      {state.message}
    </p>
  );
}
