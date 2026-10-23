"use client";

import { useActionState, useEffect, useState, type ChangeEvent } from "react";
import Cropper from "react-easy-crop";
import {
  requestEmailChangeAction,
  updateAvatarAction,
  updateContactAction,
  updateNewsletterAction,
} from "@/app/actions/account";
import { logoutAllAction } from "@/app/actions/auth";
import { AddressField } from "@/components/ui/AddressField";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { FormMessage } from "@/components/ui/FormMessage";
import { SelectField } from "@/components/ui/SelectField";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextField } from "@/components/ui/TextField";
import { cropAvatar, type CropArea } from "@/lib/avatar-image";
import { countries, DEFAULT_COUNTRY } from "@/lib/countries";
import { AVATAR_TYPES, type ContactField } from "@/lib/validation/account";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import type { PostalAddress } from "@/types";

/*
 * Formulaires du compte. Ils servent aussi aux profils des super-héros et des modérateurs : l'action
 * par défaut (espace citoyen) peut être remplacée par celle d'un autre espace.
 */

type FormAction<Field extends string> = (
  previousState: FormState<Field>,
  formData: FormData,
) => Promise<FormState<Field>>;

/**
 * Choix de la photo : aperçu, recadrage rond (glisser, molette ou flèches du clavier) et zoom.
 * Seule l'image recadrée et compressée est envoyée (voir `cropAvatar`), jamais le fichier d'origine.
 */
export function AvatarForm({
  action = updateAvatarAction,
}: { action?: FormAction<"avatar"> } = {}) {
  // Image choisie (URL locale `blob:`), `null` tant qu'aucun fichier n'est sélectionné.
  const [source, setSource] = useState<string | null>(null);
  const [crop, setCrop] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [area, setArea] = useState<CropArea | null>(null);
  const [fileError, setFileError] = useState<string>();

  const [state, formAction] = useActionState(
    async (previousState: FormState<"avatar">, formData: FormData) => {
      if (!source || !area) {
        return { status: "error", fieldErrors: { avatar: "Veuillez choisir une image." } } as const;
      }
      try {
        formData.set("avatar", await cropAvatar(source, area));
      } catch {
        return {
          status: "error",
          fieldErrors: { avatar: "Impossible de lire cette image." },
        } as const;
      }
      const result = await action(previousState, formData);
      if (result.status === "success") setSource(null);
      return result;
    },
    idleFormState as FormState<"avatar">,
  );

  // Libère l'image locale quand elle est remplacée, envoyée ou que le formulaire disparaît.
  useEffect(() => (source ? () => URL.revokeObjectURL(source) : undefined), [source]);

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    // Vidé aussitôt : choisir à nouveau le même fichier après « Annuler » redéclenche onChange.
    event.target.value = "";
    if (!file) return;
    if (!AVATAR_TYPES.includes(file.type)) {
      setFileError("Formats acceptés : JPEG, PNG ou WebP.");
      return;
    }
    setFileError(undefined);
    setCrop({ x: 0, y: 0 });
    setZoom(1);
    setSource(URL.createObjectURL(file));
  };

  return (
    <form action={formAction} className="space-y-3">
      <TextField
        label={source ? "Choisir une autre photo" : "Changer de photo"}
        hint="JPEG, PNG ou WebP. Vous pourrez ensuite la recadrer."
        // Sans `name` : le fichier d'origine n'est pas envoyé, seulement l'image recadrée.
        type="file"
        accept={AVATAR_TYPES.join(",")}
        onChange={handleFileChange}
        error={fileError ?? state.fieldErrors.avatar}
        className="[&_input]:py-2.5 [&_input]:file:mr-3 [&_input]:file:rounded-full [&_input]:file:border-0 [&_input]:file:bg-surface-raised [&_input]:file:px-3 [&_input]:file:py-1 [&_input]:file:text-foreground"
      />
      {source && (
        <>
          <div className="relative aspect-square w-full overflow-hidden border-2 border-edge bg-surface-raised">
            <Cropper
              image={source}
              crop={crop}
              zoom={zoom}
              aspect={1}
              cropShape="round"
              showGrid={false}
              onCropChange={setCrop}
              onZoomChange={setZoom}
              onCropComplete={(_, pixels) => setArea(pixels)}
            />
          </div>
          <label className="flex items-center gap-3 text-sm font-medium">
            Zoom
            <input
              type="range"
              min={1}
              max={3}
              step={0.01}
              value={zoom}
              onChange={(event) => setZoom(Number(event.target.value))}
              className="w-full accent-primary"
            />
          </label>
        </>
      )}
      <FormMessage state={state} />
      {source && (
        <div className="flex flex-wrap gap-3">
          <SubmitButton variant="secondary" pendingLabel="Envoi…">
            Enregistrer la photo
          </SubmitButton>
          <button type="button" onClick={() => setSource(null)} className={buttonClasses("ghost")}>
            Annuler
          </button>
        </div>
      )}
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
      <AddressField
        label="Numéro et nom de la rue"
        hint="Commencez à saisir l'adresse, puis choisissez-la dans la liste."
        name="street"
        autoComplete="address-line1"
        required
        defaultValue={value("street", address?.street ?? "")}
        error={errors.street}
        fill={{ postalCode: "postalCode", city: "city", country: "country" }}
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
        hint="Un lien de confirmation y sera envoyé. Elle remplacera l'adresse actuelle une fois confirmée, puis validée par la modération."
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

export function NewsletterSettingsForm({
  subscribed,
  action = updateNewsletterAction,
}: {
  subscribed: boolean;
  action?: FormAction<string>;
}) {
  const [state, formAction] = useActionState(action, idleFormState);

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

/** « Me déconnecter partout » : ferme toutes les sessions du compte, celle-ci comprise. */
export function LogoutAllForm() {
  const [state, formAction] = useActionState(logoutAllAction, idleFormState);

  return (
    <form action={formAction} className="space-y-4">
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Déconnexion…">
        Me déconnecter partout
      </SubmitButton>
    </form>
  );
}
