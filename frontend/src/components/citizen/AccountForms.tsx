"use client";

import { useActionState } from "react";
import {
  requestEmailChangeAction,
  updateAvatarAction,
  updateContactAction,
  updateNewsletterAction,
} from "@/app/actions/account";
import { FormMessage } from "@/components/ui/FormMessage";
import { SelectField } from "@/components/ui/SelectField";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextField } from "@/components/ui/TextField";
import { countries, DEFAULT_COUNTRY } from "@/lib/countries";
import type { ContactField } from "@/lib/validation/account";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import type { PostalAddress } from "@/types";

/*
 * Formulaires du compte. Ils servent aussi au profil des super-héros : l'action par défaut
 * (espace citoyen) peut être remplacée par celle d'un autre espace.
 */

type FormAction<Field extends string> = (
  previousState: FormState<Field>,
  formData: FormData,
) => Promise<FormState<Field>>;

export function AvatarForm({
  action = updateAvatarAction,
}: { action?: FormAction<"avatar"> } = {}) {
  const [state, formAction] = useActionState(action, idleFormState as FormState<"avatar">);

  return (
    <form action={formAction} className="space-y-3">
      <TextField
        label="Changer de photo"
        hint="JPEG, PNG ou WebP, 2 Mo maximum."
        name="avatar"
        type="file"
        accept="image/jpeg,image/png,image/webp"
        required
        error={state.fieldErrors.avatar}
        className="[&_input]:py-2.5 [&_input]:file:mr-3 [&_input]:file:rounded-full [&_input]:file:border-0 [&_input]:file:bg-surface-raised [&_input]:file:px-3 [&_input]:file:py-1 [&_input]:file:text-foreground"
      />
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Envoi…">
        Mettre à jour la photo
      </SubmitButton>
    </form>
  );
}

interface ContactFormProps {
  contact: {
    mobilePhone: string;
    /** `null` si aucune adresse n'est encore connue (profil de super-héros). */
    address: (PostalAddress & { complement?: string | null }) | null;
  };
  action?: FormAction<ContactField>;
  /** Affiche le champ « Complément d'adresse » (absent du profil des super-héros). */
  withComplement?: boolean;
}

export function ContactForm({
  contact,
  action = updateContactAction,
  withComplement = true,
}: ContactFormProps) {
  const [state, formAction] = useActionState(action, idleFormState as FormState<ContactField>);
  const address = contact.address;
  const errors = state.fieldErrors;
  // Après une erreur, on réaffiche la saisie ; sinon les valeurs enregistrées.
  const value = (field: ContactField, saved: string) => state.values?.[field] ?? saved;

  return (
    <form action={formAction} noValidate className="space-y-5">
      <TextField
        label="Numéro de portable"
        name="mobilePhone"
        type="tel"
        inputMode="tel"
        autoComplete="tel"
        required
        defaultValue={value("mobilePhone", contact.mobilePhone)}
        error={errors.mobilePhone}
      />
      <TextField
        label="Numéro et nom de la rue"
        name="street"
        autoComplete="address-line1"
        required
        defaultValue={value("street", address?.street ?? "")}
        error={errors.street}
      />
      {withComplement && (
        <TextField
          label="Complément d'adresse (facultatif)"
          hint="Bâtiment, étage, code d'accès…"
          name="complement"
          autoComplete="address-line2"
          defaultValue={value("complement", address?.complement ?? "")}
          error={errors.complement}
        />
      )}
      <div className="grid gap-5 sm:grid-cols-[10rem_1fr]">
        <TextField
          label="Code postal"
          name="postalCode"
          autoComplete="postal-code"
          required
          defaultValue={value("postalCode", address?.postalCode ?? "")}
          error={errors.postalCode}
        />
        <TextField
          label="Ville"
          name="city"
          autoComplete="address-level2"
          required
          defaultValue={value("city", address?.city ?? "")}
          error={errors.city}
        />
      </div>
      <SelectField
        label="Pays"
        name="country"
        autoComplete="country"
        required
        defaultValue={value("country", address?.country ?? DEFAULT_COUNTRY)}
        error={errors.country}
      >
        {countries.map((country) => (
          <option key={country.code} value={country.code}>
            {country.name}
          </option>
        ))}
      </SelectField>
      <FormMessage state={state} />
      <SubmitButton pendingLabel="Enregistrement…">Enregistrer mes coordonnées</SubmitButton>
    </form>
  );
}

export function EmailChangeForm({
  action = requestEmailChangeAction,
}: { action?: FormAction<"email"> } = {}) {
  const [state, formAction] = useActionState(action, idleFormState as FormState<"email">);

  return (
    <form action={formAction} noValidate className="space-y-4">
      <TextField
        label="Nouvelle adresse email"
        hint="Elle remplacera l'adresse actuelle après validation par la modération."
        name="email"
        type="email"
        inputMode="email"
        autoComplete="email"
        autoCapitalize="none"
        spellCheck={false}
        required
        defaultValue={state.values?.email ?? ""}
        error={state.fieldErrors.email}
      />
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Envoi…">
        Demander la modification
      </SubmitButton>
    </form>
  );
}

export function NewsletterSettingsForm({ subscribed }: { subscribed: boolean }) {
  const [state, formAction] = useActionState(updateNewsletterAction, idleFormState);

  return (
    <form action={formAction} className="space-y-4">
      <label className="flex cursor-pointer items-start gap-3">
        <input
          type="checkbox"
          name="newsletter"
          role="switch"
          defaultChecked={subscribed}
          className="peer sr-only"
        />
        <span
          aria-hidden="true"
          className="relative mt-0.5 inline-flex h-6 w-11 shrink-0 rounded-full border border-border bg-surface-raised transition-colors peer-checked:border-primary peer-checked:bg-primary peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-primary after:absolute after:top-0.5 after:left-0.5 after:size-4.5 after:rounded-full after:bg-foreground after:transition-transform peer-checked:after:translate-x-5 motion-reduce:transition-none motion-reduce:after:transition-none"
        />
        <span>
          <span className="block font-medium">Recevoir la newsletter</span>
          <span className="text-sm text-muted">
            Interventions marquantes, nouveaux héros et conseils de sécurité, une fois par mois.
          </span>
        </span>
      </label>
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Enregistrement…">
        Enregistrer ma préférence
      </SubmitButton>
    </form>
  );
}
