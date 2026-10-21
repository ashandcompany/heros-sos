"use client";

import { useActionState, useId } from "react";
import { saveVillainAction } from "@/app/actions/admin";
import { FormMessage } from "@/components/ui/FormMessage";
import { SelectField } from "@/components/ui/SelectField";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { TextField } from "@/components/ui/TextField";
import { threatLevelLabels, villainStatusLabels } from "@/lib/format";
import {
  VILLAIN_ALIAS_MAX_LENGTH,
  VILLAIN_DESCRIPTION_MAX_LENGTH,
  VILLAIN_NAME_MAX_LENGTH,
  VILLAIN_SECTOR_MAX_LENGTH,
  parseVillainTypeIds,
  type VillainField,
} from "@/lib/validation/admin";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import type { IncidentTypeDefinition, Villain } from "@/types";

/** Création (sans `villain`) ou modification d'une fiche de super-vilain par la modération. */
export function VillainForm({
  villain,
  types,
}: {
  villain?: Villain;
  /** Référentiel des types d'incident, par libellé : les spécialités possibles. */
  types: Array<Pick<IncidentTypeDefinition, "id" | "label">>;
}) {
  const [state, formAction] = useActionState(
    saveVillainAction,
    idleFormState as FormState<VillainField>,
  );
  const errors = state.fieldErrors;
  // Après une erreur, on réaffiche la saisie ; sinon les valeurs enregistrées.
  const value = (field: VillainField, saved: string | undefined) =>
    state.values?.[field] ?? saved ?? "";
  const checkedTypeIds = new Set(
    state.values?.typeIds !== undefined
      ? parseVillainTypeIds(state.values.typeIds)
      : (villain?.typeIds ?? []),
  );
  const typesErrorId = useId();

  return (
    <form action={formAction} noValidate className="space-y-5">
      {villain && <input type="hidden" name="id" value={villain.id} />}

      <TextField
        label="Alias"
        name="alias"
        hint="Nom sous lequel le super-vilain est connu ; il doit être unique."
        required
        maxLength={VILLAIN_ALIAS_MAX_LENGTH}
        defaultValue={value("alias", villain?.alias)}
        error={errors.alias}
      />

      <div className="grid gap-5 sm:grid-cols-2">
        <TextField
          label="Prénom (facultatif)"
          name="firstName"
          maxLength={VILLAIN_NAME_MAX_LENGTH}
          defaultValue={value("firstName", villain?.firstName)}
          error={errors.firstName}
        />
        <TextField
          label="Nom (facultatif)"
          name="lastName"
          maxLength={VILLAIN_NAME_MAX_LENGTH}
          defaultValue={value("lastName", villain?.lastName)}
          error={errors.lastName}
        />
      </div>

      <TextField
        label="Secteur principal (facultatif)"
        name="mainSector"
        hint="Ville ou quartier où il sévit le plus souvent."
        maxLength={VILLAIN_SECTOR_MAX_LENGTH}
        defaultValue={value("mainSector", villain?.mainSector)}
        error={errors.mainSector}
      />

      <TextAreaField
        label="Mode opératoire (facultatif)"
        name="description"
        hint="Affiché sur sa fiche publique de l'encyclopédie."
        maxLength={VILLAIN_DESCRIPTION_MAX_LENGTH}
        defaultValue={value("description", villain?.description)}
        error={errors.description}
      />

      <div className="grid gap-5 sm:grid-cols-2">
        <SelectField
          label="Niveau de menace"
          name="threatLevel"
          required
          defaultValue={value("threatLevel", villain?.threatLevel ?? "faible")}
          error={errors.threatLevel}
        >
          {Object.entries(threatLevelLabels).map(([level, label]) => (
            <option key={level} value={level}>
              {label}
            </option>
          ))}
        </SelectField>
        <SelectField
          label="Statut"
          name="status"
          required
          defaultValue={value("status", villain?.status ?? "en-liberte")}
          error={errors.status}
        >
          {Object.entries(villainStatusLabels).map(([status, label]) => (
            <option key={status} value={status}>
              {label}
            </option>
          ))}
        </SelectField>
      </div>

      <fieldset aria-describedby={errors.typeIds ? typesErrorId : undefined}>
        <legend className="mb-2 text-sm font-medium">Spécialités (facultatif)</legend>
        {types.length === 0 ? (
          <p className="text-sm text-muted">
            Aucun type d&apos;incident dans le référentiel : aucune spécialité à proposer.
          </p>
        ) : (
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {types.map((type) => (
              <label key={type.id} className="flex items-center gap-3 text-sm">
                <input
                  type="checkbox"
                  name="typeIds"
                  value={type.id}
                  defaultChecked={checkedTypeIds.has(type.id)}
                  className="size-5 accent-primary"
                />
                {type.label}
              </label>
            ))}
          </div>
        )}
        {errors.typeIds && (
          <p id={typesErrorId} className="mt-2 text-sm text-danger">
            {errors.typeIds}
          </p>
        )}
      </fieldset>

      <FormMessage state={state} />
      <SubmitButton pendingLabel="Enregistrement…">
        {villain ? "Enregistrer la fiche" : "Créer la fiche"}
      </SubmitButton>
    </form>
  );
}
