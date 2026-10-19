"use client";

import { useActionState } from "react";
import { updateIncidentAction } from "@/app/actions/admin";
import { FormMessage } from "@/components/ui/FormMessage";
import { SelectField } from "@/components/ui/SelectField";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { TextField } from "@/components/ui/TextField";
import { incidentStatusLabels, incidentTypeLabels } from "@/lib/format";
import type { IncidentEditField } from "@/lib/validation/admin";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import { DESCRIPTION_MAX_LENGTH, TITLE_MAX_LENGTH } from "@/lib/validation/incident";
import type { Incident, SuperHeroSummary, VillainSummary } from "@/types";

/** Correction des informations d'un incident par un modérateur. */
export function IncidentEditForm({
  incident,
  villains,
  heroes,
}: {
  incident: Incident;
  villains: VillainSummary[];
  heroes: SuperHeroSummary[];
}) {
  const [state, formAction] = useActionState(
    updateIncidentAction,
    idleFormState as FormState<IncidentEditField>,
  );
  const errors = state.fieldErrors;
  // Après une erreur, on réaffiche la saisie ; sinon les valeurs enregistrées.
  const value = (field: IncidentEditField, saved: string | undefined) =>
    state.values?.[field] ?? saved ?? "";

  return (
    <form action={formAction} noValidate className="space-y-5">
      <input type="hidden" name="id" value={incident.id} />

      <TextField
        label="Titre"
        name="title"
        required
        maxLength={TITLE_MAX_LENGTH}
        defaultValue={value("title", incident.title)}
        error={errors.title}
      />

      <div className="grid gap-5 sm:grid-cols-2">
        <SelectField
          label="Type d'incident"
          name="type"
          required
          defaultValue={value("type", incident.type)}
          error={errors.type}
        >
          {Object.entries(incidentTypeLabels).map(([type, label]) => (
            <option key={type} value={type}>
              {label}
            </option>
          ))}
        </SelectField>
        <SelectField
          label="Statut"
          name="status"
          required
          defaultValue={value("status", incident.status)}
          error={errors.status}
        >
          {Object.entries(incidentStatusLabels).map(([status, label]) => (
            <option key={status} value={status}>
              {label}
            </option>
          ))}
        </SelectField>
      </div>

      <TextAreaField
        label="Description"
        name="description"
        rows={5}
        maxLength={DESCRIPTION_MAX_LENGTH}
        defaultValue={value("description", incident.description)}
        error={errors.description}
      />

      <div className="grid gap-5 sm:grid-cols-[1fr_12rem]">
        <TextField
          label="Lieu précis"
          name="address"
          defaultValue={value("address", incident.location.address)}
          error={errors.address}
        />
        <TextField
          label="Ville"
          name="city"
          required
          defaultValue={value("city", incident.location.city)}
          error={errors.city}
        />
      </div>

      <div className="grid gap-5 sm:grid-cols-2">
        <SelectField
          label="Super-vilain impliqué"
          name="villainId"
          defaultValue={value("villainId", incident.villain?.id)}
          error={errors.villainId}
        >
          <option value="">Non identifié</option>
          {villains.map((villain) => (
            <option key={villain.id} value={villain.id}>
              {villain.alias}
            </option>
          ))}
        </SelectField>
        <SelectField
          label="Super-héros intervenant"
          name="heroId"
          defaultValue={value("heroId", incident.hero?.id)}
          error={errors.heroId}
        >
          <option value="">Aucun (incident non pris en charge)</option>
          {heroes.map((hero) => (
            <option key={hero.id} value={hero.id}>
              {hero.alias}
            </option>
          ))}
        </SelectField>
      </div>

      <FormMessage state={state} />
      <SubmitButton pendingLabel="Enregistrement…">Enregistrer les modifications</SubmitButton>
    </form>
  );
}
