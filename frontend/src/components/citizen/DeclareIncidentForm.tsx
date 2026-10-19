"use client";

import { useActionState, useEffect, useRef, type FormEvent } from "react";
import { declareIncidentAction } from "@/app/actions/incidents";
import { FormMessage } from "@/components/ui/FormMessage";
import { RadioGroup } from "@/components/ui/RadioGroup";
import { SelectField } from "@/components/ui/SelectField";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { TextField } from "@/components/ui/TextField";
import { incidentTypeLabels } from "@/lib/format";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import {
  declarationFields,
  DESCRIPTION_MAX_LENGTH,
  DESCRIPTION_MIN_LENGTH,
  reporterRoleOptions,
  TITLE_MAX_LENGTH,
  type DeclarationField,
} from "@/lib/validation/incident";
import type { VillainSummary } from "@/types";

export function DeclareIncidentForm({ villains }: { villains: VillainSummary[] }) {
  const [state, formAction] = useActionState(
    declareIncidentAction,
    idleFormState as FormState<DeclarationField>,
  );
  const formRef = useRef<HTMLFormElement>(null);
  const occurredAtIsoRef = useRef<HTMLInputElement>(null);
  const errors = state.fieldErrors;
  const values = state.values ?? {};

  // Après une réponse en erreur : focus sur le premier champ invalide.
  useEffect(() => {
    if (state.status !== "error") return;
    const field = declarationFields.find((name) => state.fieldErrors[name]);
    if (field) formRef.current?.querySelector<HTMLElement>(`[name="${field}"]`)?.focus();
  }, [state]);

  // Convertit la date locale saisie en ISO (fuseau du navigateur) avant l'envoi.
  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    const local = new FormData(event.currentTarget).get("occurredAt");
    const date = typeof local === "string" && local ? new Date(local) : null;
    if (occurredAtIsoRef.current) {
      occurredAtIsoRef.current.value =
        date && !Number.isNaN(date.getTime()) ? date.toISOString() : "";
    }
  };

  return (
    <form
      ref={formRef}
      action={formAction}
      onSubmit={handleSubmit}
      noValidate
      className="space-y-5"
    >
      <input ref={occurredAtIsoRef} type="hidden" name="occurredAtIso" />

      <RadioGroup
        legend="Vous êtes"
        name="reporterRole"
        options={reporterRoleOptions}
        defaultValue={values.reporterRole}
        error={errors.reporterRole}
        required
      />

      <SelectField
        label="Type d'incident"
        name="type"
        required
        defaultValue={values.type ?? ""}
        error={errors.type}
      >
        <option value="" disabled>
          Choisir…
        </option>
        {Object.entries(incidentTypeLabels).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </SelectField>

      <TextField
        label="Titre"
        hint="Résumez l'incident en quelques mots."
        name="title"
        required
        maxLength={TITLE_MAX_LENGTH}
        defaultValue={values.title}
        error={errors.title}
      />

      <TextAreaField
        label="Description"
        hint={`Ce que vous voyez, le nombre de personnes impliquées, les dangers immédiats… (${DESCRIPTION_MIN_LENGTH} à ${DESCRIPTION_MAX_LENGTH} caractères)`}
        name="description"
        required
        rows={5}
        maxLength={DESCRIPTION_MAX_LENGTH}
        defaultValue={values.description}
        error={errors.description}
      />

      <div className="grid gap-5 sm:grid-cols-[1fr_12rem]">
        <TextField
          label="Lieu précis"
          hint="Adresse, nom d'un commerce, repère…"
          name="address"
          autoComplete="street-address"
          required
          defaultValue={values.address}
          error={errors.address}
        />
        <TextField
          label="Ville"
          name="city"
          autoComplete="address-level2"
          required
          defaultValue={values.city}
          error={errors.city}
        />
      </div>

      <TextField
        label="Date et heure des faits (facultatif)"
        hint="Laissez vide si l'incident est en train de se produire."
        name="occurredAt"
        type="datetime-local"
        defaultValue={values.occurredAt}
        error={errors.occurredAt}
      />

      <SelectField
        label="Super-vilain impliqué"
        name="villainId"
        defaultValue={values.villainId ?? ""}
        error={errors.villainId}
      >
        <option value="">Je ne sais pas</option>
        {villains.map((villain) => (
          <option key={villain.id} value={villain.id}>
            {villain.alias}
          </option>
        ))}
      </SelectField>

      <FormMessage state={state} />
      <SubmitButton pendingLabel="Envoi…" className="w-full sm:w-auto">
        Déclarer l&apos;incident
      </SubmitButton>
    </form>
  );
}
