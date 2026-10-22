"use client";

import { useActionState, useCallback, useEffect, useRef, type FormEvent } from "react";
import type { DeclareIncidentFormAction } from "@/app/actions/incidents";
import { useOfflineAction } from "@/components/offline/useOfflineAction";
import { AddressField } from "@/components/ui/AddressField";
import { FormMessage } from "@/components/ui/FormMessage";
import { RadioGroup } from "@/components/ui/RadioGroup";
import { SelectField } from "@/components/ui/SelectField";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { TextField } from "@/components/ui/TextField";
import { commands } from "@/lib/offline/commands";
import type { BuiltCommand } from "@/lib/offline/forms";
import { hasFieldErrors, idleFormState, type FormState } from "@/lib/validation/form-state";
import {
  declarationFields,
  DESCRIPTION_MAX_LENGTH,
  DESCRIPTION_MIN_LENGTH,
  readDeclarationValues,
  reporterRoleOptions,
  TITLE_MAX_LENGTH,
  toDeclaration,
  validateDeclaration,
  type DeclarationField,
} from "@/lib/validation/incident";
import type { IncidentTypeDefinition, VillainSummary } from "@/types";

interface DeclareIncidentFormProps {
  villains: VillainSummary[];
  /** Types du référentiel, par libellé. */
  types: Array<Pick<IncidentTypeDefinition, "id" | "label">>;
  /** Action de déclaration : citoyen (espace personnel) ou modérateur (back-office). */
  action: DeclareIncidentFormAction;
  /** Demande « témoin ou victime » (seulement sans l'API, qui ne le connaît pas). */
  askReporterRole: boolean;
}

export function DeclareIncidentForm({
  villains,
  types,
  action,
  askReporterRole,
}: DeclareIncidentFormProps) {
  // Hors ligne (ou réseau coupé pendant l'envoi), la déclaration est gardée sur l'appareil et partira au retour du
  // réseau : même validation que côté serveur, sur les types et vilains que le formulaire affiche.
  const buildOffline = useCallback(
    (formData: FormData): BuiltCommand<FormState<DeclarationField>> => {
      const values = readDeclarationValues(formData);
      const occurredAtIso = String(formData.get("occurredAtIso") ?? "");
      const validated = { ...values, occurredAt: occurredAtIso || values.occurredAt };
      const fieldErrors = validateDeclaration(validated, {
        villainIds: villains.map(({ id }) => id),
        typeIds: types.map(({ id }) => id),
        askReporterRole,
      });
      if (hasFieldErrors(fieldErrors)) {
        return { ok: false, state: { status: "error", fieldErrors, values } };
      }
      const declaration = toDeclaration(validated);
      const draft = commands.declareIncident({
        id: String(formData.get("incidentId") ?? ""),
        typeId: declaration.typeId,
        title: declaration.title,
        description: declaration.description,
        citizensRescued: 0,
        // Le géocodage demande le réseau : l'incident arrive sans coordonnées.
        location: {
          address: declaration.location.address,
          city: declaration.location.city,
          latitude: null,
          longitude: null,
        },
        occurredAt: declaration.occurredAt,
        villainIds: declaration.villainId ? [declaration.villainId] : [],
      });
      if (!draft) {
        return {
          ok: false,
          state: { status: "error", message: "Requête invalide.", fieldErrors: {}, values },
        };
      }
      return { ok: true, draft, message: "Incident enregistré sur cet appareil." };
    },
    [villains, types, askReporterRole],
  );
  const submit = useOfflineAction(action, buildOffline);
  const [state, formAction] = useActionState(submit, idleFormState as FormState<DeclarationField>);
  const formRef = useRef<HTMLFormElement>(null);
  const occurredAtIsoRef = useRef<HTMLInputElement>(null);
  const incidentIdRef = useRef<HTMLInputElement>(null);
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
    // Un identifiant par envoi, choisi par l'appareil : la même déclaration rejouée ne crée pas deux incidents.
    if (incidentIdRef.current) incidentIdRef.current.value = crypto.randomUUID();
  };

  // Référentiel vide (aucun type créé par la modération) : rien ne peut être déclaré.
  if (types.length === 0) {
    return (
      <p
        role="status"
        className="rounded-2xl border border-dashed border-border p-5 text-sm text-muted"
      >
        La déclaration d&apos;incident est momentanément indisponible : aucun type d&apos;incident
        n&apos;a encore été défini par le Bureau.
      </p>
    );
  }

  return (
    <form
      ref={formRef}
      action={formAction}
      onSubmit={handleSubmit}
      noValidate
      className="space-y-5"
    >
      <input ref={occurredAtIsoRef} type="hidden" name="occurredAtIso" />
      <input ref={incidentIdRef} type="hidden" name="incidentId" />

      {askReporterRole && (
        <RadioGroup
          legend="Vous êtes"
          name="reporterRole"
          options={reporterRoleOptions}
          defaultValue={values.reporterRole}
          error={errors.reporterRole}
          required
        />
      )}

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
        {types.map((type) => (
          <option key={type.id} value={type.id}>
            {type.label}
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
        <AddressField
          kind="place"
          label="Lieu précis"
          hint="Adresse, nom d'un commerce, gare… Choisissez une suggestion pour remplir la ville."
          name="address"
          autoComplete="off"
          required
          defaultValue={values.address}
          error={errors.address}
          fill={{ city: "city" }}
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
        hint={
          villains.length === 0
            ? "Aucun super-vilain n'est encore répertorié : précisez-le dans la description si vous le connaissez."
            : undefined
        }
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
