"use client";

import { useActionState, useEffect, useRef, useState, type FormEvent } from "react";
import { signupAction, type SignupFormState } from "@/app/actions/signup";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { PasswordField } from "@/components/ui/PasswordField";
import { RadioGroup } from "@/components/ui/RadioGroup";
import { SelectField } from "@/components/ui/SelectField";
import { TextField } from "@/components/ui/TextField";
import { countries } from "@/lib/countries";
import {
  civilityOptions,
  emptySignupValues,
  firstInvalidField,
  PASSWORD_RULES,
  readSignupValues,
  signupFields,
  signupSteps,
  stepOf,
  validateSignup,
  type SignupErrors,
  type SignupField,
  type SignupStep,
} from "@/lib/validation/signup";

const initialState: SignupFormState = { step: 1, values: emptySignupValues, fieldErrors: {} };

/** Élément à focaliser après un rendu : titre de l'étape ou premier champ en erreur. */
type FocusRequest = { target: SignupField | "heading" };

function StepIndicator({ current }: { current: SignupStep }) {
  return (
    <ol aria-label="Étapes de l'inscription" className="mb-8 grid grid-cols-2 gap-3">
      {signupSteps.map(({ step, title }) => {
        const isCurrent = step === current;
        const isDone = step < current;
        return (
          <li key={step} aria-current={isCurrent ? "step" : undefined}>
            <span
              aria-hidden="true"
              className={cn(
                "mb-3 block h-1 rounded-full",
                isCurrent || isDone ? "bg-primary" : "bg-border",
              )}
            />
            <span className="flex items-center gap-2 text-sm">
              <span
                aria-hidden="true"
                className={cn(
                  "inline-flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-bold",
                  isCurrent && "bg-primary text-primary-foreground",
                  isDone && "bg-success/15 text-success",
                  !isCurrent && !isDone && "border border-border text-muted",
                )}
              >
                {isDone ? "✓" : step}
              </span>
              <span className={isCurrent ? "font-semibold" : "text-muted"}>
                <span className="sr-only">Étape {step} : </span>
                {title}
                {isDone && <span className="sr-only"> (terminée)</span>}
              </span>
            </span>
          </li>
        );
      })}
    </ol>
  );
}

export function SignupForm() {
  const [state, formAction, isPending] = useActionState(signupAction, initialState);
  const [step, setStep] = useState<SignupStep>(1);
  const [clientErrors, setClientErrors] = useState<SignupErrors>({});
  const [focusRequest, setFocusRequest] = useState<FocusRequest | null>(null);
  const [handledState, setHandledState] = useState(state);
  const formRef = useRef<HTMLFormElement>(null);
  const headingRef = useRef<HTMLHeadingElement>(null);

  // Nouvelle réponse du serveur : on affiche l'étape du premier champ en erreur.
  if (state !== handledState) {
    setHandledState(state);
    setStep(state.step);
    setClientErrors({});
    setFocusRequest({ target: firstInvalidField(state.fieldErrors) ?? "heading" });
  }

  useEffect(() => {
    if (!focusRequest) return;
    if (focusRequest.target === "heading") {
      headingRef.current?.focus();
    } else {
      formRef.current?.querySelector<HTMLElement>(`[name="${focusRequest.target}"]`)?.focus();
    }
  }, [focusRequest]);

  // Les erreurs client (re)validées remplacent celles du serveur, y compris quand elles valent undefined.
  const errors: SignupErrors = { ...state.fieldErrors, ...clientErrors };
  const values = state.values;
  const currentStep = signupSteps[step - 1];

  const validateFields = (fields: readonly SignupField[]) => {
    if (!formRef.current) return {};
    const fieldErrors = validateSignup(readSignupValues(new FormData(formRef.current)), fields);
    setClientErrors((previous) => ({ ...previous, ...fieldErrors }));
    return fieldErrors;
  };

  const goToStep = (next: SignupStep) => {
    setStep(next);
    setFocusRequest({ target: "heading" });
  };

  const handleContinue = () => {
    const invalidField = firstInvalidField(validateFields(signupSteps[0].fields));
    if (invalidField) {
      setFocusRequest({ target: invalidField });
    } else {
      goToStep(2);
    }
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    // Entrée à l'étape 1 : équivaut à « Continuer ».
    if (step === 1) {
      event.preventDefault();
      handleContinue();
      return;
    }

    const invalidField = firstInvalidField(validateFields(signupFields));
    if (invalidField) {
      event.preventDefault();
      setStep(stepOf(invalidField));
      setFocusRequest({ target: invalidField });
    }
  };

  return (
    <div>
      <StepIndicator current={step} />

      <form ref={formRef} action={formAction} onSubmit={handleSubmit} noValidate>
        <h2
          ref={headingRef}
          tabIndex={-1}
          className="mb-6 font-display text-xl font-semibold focus:outline-none"
        >
          <span className="block text-sm font-medium text-muted">Étape {step} sur 2</span>
          {currentStep.title}
        </h2>

        <div hidden={step !== 1} className="space-y-5">
          <RadioGroup
            legend="Civilité"
            name="civility"
            options={civilityOptions}
            defaultValue={values.civility}
            error={errors.civility}
            required
          />
          <div className="grid gap-5 sm:grid-cols-2">
            <TextField
              label="Prénom"
              name="firstName"
              autoComplete="given-name"
              required
              defaultValue={values.firstName}
              error={errors.firstName}
            />
            <TextField
              label="Nom"
              name="lastName"
              autoComplete="family-name"
              required
              defaultValue={values.lastName}
              error={errors.lastName}
            />
          </div>
          <TextField
            label="Date de naissance"
            name="birthDate"
            type="date"
            autoComplete="bday"
            required
            defaultValue={values.birthDate}
            error={errors.birthDate}
          />
          <TextField
            label="Adresse email"
            hint="Elle vous servira d'identifiant de connexion."
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

        <div hidden={step !== 2} className="space-y-8">
          <fieldset className="space-y-5">
            <legend className="mb-5 font-display text-lg font-semibold">
              Informations complémentaires
            </legend>
            <TextField
              label="Numéro de portable"
              hint="Pour être joint·e lors du traitement d'un incident."
              name="mobilePhone"
              type="tel"
              inputMode="tel"
              autoComplete="tel"
              required
              defaultValue={values.mobilePhone}
              error={errors.mobilePhone}
            />
            <TextField
              label="Numéro et nom de la rue"
              name="street"
              autoComplete="address-line1"
              required
              defaultValue={values.street}
              error={errors.street}
            />
            <div className="grid gap-5 sm:grid-cols-[10rem_1fr]">
              <TextField
                label="Code postal"
                name="postalCode"
                autoComplete="postal-code"
                required
                defaultValue={values.postalCode}
                error={errors.postalCode}
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
            <SelectField
              label="Pays"
              name="country"
              autoComplete="country"
              required
              defaultValue={values.country}
              error={errors.country}
            >
              {countries.map((country) => (
                <option key={country.code} value={country.code}>
                  {country.name}
                </option>
              ))}
            </SelectField>
          </fieldset>

          <fieldset className="space-y-5">
            <legend className="mb-5 font-display text-lg font-semibold">Sécurité</legend>
            <PasswordField
              label="Mot de passe"
              hint={PASSWORD_RULES}
              name="password"
              autoComplete="new-password"
              required
              error={errors.password}
            />
            <PasswordField
              label="Confirmation du mot de passe"
              name="passwordConfirmation"
              autoComplete="new-password"
              required
              error={errors.passwordConfirmation}
            />
          </fieldset>
        </div>

        <div className="mt-8 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          {step === 1 ? (
            <button
              type="button"
              onClick={handleContinue}
              className={cn(buttonClasses("primary", "lg"), "w-full sm:w-auto")}
            >
              Continuer
            </button>
          ) : (
            <>
              <button
                type="button"
                onClick={() => goToStep(1)}
                className={cn(buttonClasses("secondary", "lg"), "w-full sm:w-auto")}
              >
                Retour
              </button>
              <button
                type="submit"
                disabled={isPending}
                className={cn(
                  buttonClasses("primary", "lg"),
                  "w-full disabled:opacity-60 sm:w-auto",
                )}
              >
                {isPending ? "Création du compte…" : "Créer mon compte"}
              </button>
            </>
          )}
        </div>

        <p role="status" className="sr-only">
          {isPending ? "Création du compte en cours" : ""}
        </p>
      </form>
    </div>
  );
}
