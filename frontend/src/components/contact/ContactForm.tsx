"use client";

import Link from "next/link";
import { useActionState, useEffect, useRef } from "react";
import { sendContactMessageAction } from "@/app/actions/contact";
import { FormMessage } from "@/components/ui/FormMessage";
import { SelectField } from "@/components/ui/SelectField";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { TextField } from "@/components/ui/TextField";
import { routes } from "@/lib/routes";
import {
  contactFields,
  contactSubjectLabels,
  HONEYPOT_FIELD,
  MESSAGE_MAX_LENGTH,
  NAME_MAX_LENGTH,
  type ContactField,
} from "@/lib/validation/contact";
import { idleFormState, type FormState } from "@/lib/validation/form-state";

/** Formulaire de contact ; remplacé par un récépissé une fois le message transmis. */
export function ContactForm() {
  const [state, formAction] = useActionState(
    sendContactMessageAction,
    idleFormState as FormState<ContactField>,
  );
  const formRef = useRef<HTMLFormElement>(null);
  const errors = state.fieldErrors;
  const values = state.values ?? {};

  // Après une réponse en erreur : focus sur le premier champ invalide.
  useEffect(() => {
    if (state.status !== "error") return;
    const field = contactFields.find((name) => state.fieldErrors[name]);
    if (field) formRef.current?.querySelector<HTMLElement>(`[name="${field}"]`)?.focus();
  }, [state]);

  if (state.status === "success") {
    return (
      <div role="status" className="paper relative border-2 border-edge p-6 pt-8">
        <p className="font-mono text-xs tracking-widest uppercase">Récépissé de dépôt</p>
        <p className="mt-3 font-display text-2xl font-black tracking-wider uppercase">
          Message reçu
        </p>
        <p className="mt-3 text-pretty text-ink/85">{state.message}</p>
        <span aria-hidden="true" className="stamp absolute top-6 right-4 text-lg">
          Reçu
        </span>
      </div>
    );
  }

  return (
    <form ref={formRef} action={formAction} noValidate className="space-y-5">
      <div className="grid gap-5 sm:grid-cols-2">
        <TextField
          label="Nom"
          name="name"
          autoComplete="name"
          maxLength={NAME_MAX_LENGTH}
          required
          defaultValue={values.name}
          error={errors.name}
        />
        <TextField
          label="Adresse email"
          hint="Pour vous répondre."
          name="email"
          type="email"
          inputMode="email"
          autoComplete="email"
          autoCapitalize="none"
          spellCheck={false}
          required
          defaultValue={values.email}
          error={errors.email}
        />
      </div>

      <SelectField
        label="Objet"
        name="subject"
        required
        defaultValue={values.subject ?? ""}
        error={errors.subject}
      >
        <option value="" disabled>
          Choisissez l&apos;objet de votre demande
        </option>
        {Object.entries(contactSubjectLabels).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </SelectField>

      <TextAreaField
        label="Message"
        hint={`${MESSAGE_MAX_LENGTH} caractères maximum. N'indiquez ni mot de passe ni information sensible.`}
        name="message"
        rows={7}
        maxLength={MESSAGE_MAX_LENGTH}
        required
        defaultValue={values.message}
        error={errors.message}
      />

      {/* Champ piège anti-robots : invisible et hors de la navigation clavier */}
      <div aria-hidden="true" className="hidden">
        <label>
          Ne pas remplir
          <input type="text" name={HONEYPOT_FIELD} tabIndex={-1} autoComplete="off" />
        </label>
      </div>

      <FormMessage state={state} />

      <p className="text-xs text-muted">
        Vos nom, email et message servent uniquement à traiter votre demande. Pour en savoir plus et
        exercer vos droits, consultez notre{" "}
        <Link href={routes.privacy} className="text-accent underline">
          politique de confidentialité
        </Link>
        .
      </p>

      <SubmitButton pendingLabel="Transmission…" className="w-full sm:w-auto">
        Envoyer le message
      </SubmitButton>
    </form>
  );
}
