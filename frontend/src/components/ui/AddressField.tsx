"use client";

import {
  useEffect,
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type ComponentPropsWithoutRef,
  type KeyboardEvent,
} from "react";
import type { AddressSearchKind, AddressSuggestion } from "@/lib/geocoding";
import { cn } from "./cn";

const DEBOUNCE_MS = 300;
const MIN_QUERY_LENGTH = 3;

/** Champs du même formulaire remplis quand une suggestion est choisie (valeur : attribut `name`). */
type FillTargets = Partial<Record<"postalCode" | "city" | "country", string>>;

interface AddressFieldProps extends Omit<
  ComponentPropsWithoutRef<"input">,
  "children" | "type" | "onChange"
> {
  label: string;
  hint?: string;
  error?: string;
  /** `postal` : adresse postale (numéro et rue). `place` : tout lieu (commerce, gare…). */
  kind?: AddressSearchKind;
  fill?: FillTargets;
}

/** Remplit un champ du formulaire, sans toucher à une liste dont la valeur n'existe pas. */
function setFormValue(form: HTMLFormElement, name: string, value: string) {
  const element = form.elements.namedItem(name);
  if (!value) return;
  if (element instanceof HTMLSelectElement) {
    if (Array.from(element.options).some((option) => option.value === value)) {
      element.value = value;
    }
  } else if (element instanceof HTMLInputElement) {
    element.value = value;
  }
}

/**
 * Champ d'adresse avec suggestions issues d'OpenStreetMap (route /api/adresses).
 * Motif « combobox » ARIA 1.2 : flèches haut/bas, Entrée pour choisir, Échap pour fermer.
 * La saisie libre reste possible : les suggestions ne sont qu'une aide.
 */
export function AddressField({
  label,
  hint,
  error,
  kind = "postal",
  fill,
  id,
  className,
  onBlur,
  onKeyDown,
  ...inputProps
}: AddressFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const listboxId = `${inputId}-suggestions`;
  const hintId = `${inputId}-hint`;
  const errorId = `${inputId}-error`;
  const describedBy = [hint && hintId, error && errorId].filter(Boolean).join(" ") || undefined;

  const inputRef = useRef<HTMLInputElement>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const requestRef = useRef<AbortController | null>(null);

  const [suggestions, setSuggestions] = useState<AddressSuggestion[]>([]);
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [status, setStatus] = useState("");

  useEffect(
    () => () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      requestRef.current?.abort();
    },
    [],
  );

  const close = () => {
    setOpen(false);
    setActiveIndex(-1);
  };

  const search = async (query: string) => {
    requestRef.current?.abort();
    const controller = new AbortController();
    requestRef.current = controller;

    try {
      const params = new URLSearchParams({ q: query, type: kind });
      const response = await fetch(`/api/adresses?${params}`, { signal: controller.signal });
      const { suggestions: results = [] } = (await response.json()) as {
        suggestions?: AddressSuggestion[];
      };
      setSuggestions(results);
      setActiveIndex(-1);
      setOpen(results.length > 0);
      setStatus(
        results.length === 0
          ? ""
          : `${results.length} suggestion${results.length > 1 ? "s" : ""}. Flèches haut et bas pour parcourir, Entrée pour choisir.`,
      );
    } catch (cause) {
      // Requête annulée par une frappe plus récente, ou service indisponible : saisie libre.
      if (!(cause instanceof DOMException && cause.name === "AbortError")) close();
    }
  };

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    const query = event.target.value.trim();
    if (timerRef.current) clearTimeout(timerRef.current);

    if (query.length < MIN_QUERY_LENGTH) {
      requestRef.current?.abort();
      setSuggestions([]);
      close();
      return;
    }
    timerRef.current = setTimeout(() => void search(query), DEBOUNCE_MS);
  };

  const choose = (suggestion: AddressSuggestion) => {
    const input = inputRef.current;
    if (!input) return;

    input.value = kind === "postal" ? suggestion.street : suggestion.label;
    if (input.form && fill) {
      if (fill.postalCode) setFormValue(input.form, fill.postalCode, suggestion.postalCode);
      if (fill.city) setFormValue(input.form, fill.city, suggestion.city);
      if (fill.country) setFormValue(input.form, fill.country, suggestion.country);
    }
    setStatus(`Adresse choisie : ${suggestion.label}, ${suggestion.context}.`);
    close();
    input.focus();
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    onKeyDown?.(event);
    if (event.defaultPrevented) return;

    switch (event.key) {
      case "ArrowDown":
        if (suggestions.length === 0) return;
        event.preventDefault();
        setOpen(true);
        setActiveIndex((index) => (index + 1) % suggestions.length);
        break;
      case "ArrowUp":
        if (suggestions.length === 0) return;
        event.preventDefault();
        setOpen(true);
        setActiveIndex((index) => (index <= 0 ? suggestions.length - 1 : index - 1));
        break;
      case "Enter":
        // Sans suggestion active, Entrée garde son rôle : envoyer le formulaire.
        if (open && activeIndex >= 0) {
          event.preventDefault();
          choose(suggestions[activeIndex]);
        }
        break;
      case "Escape":
        if (open) {
          event.preventDefault();
          close();
        }
        break;
    }
  };

  const activeId = open && activeIndex >= 0 ? `${listboxId}-${activeIndex}` : undefined;

  return (
    <div className={className}>
      <label htmlFor={inputId} className="mb-2 block text-sm font-medium">
        {label}
      </label>
      <div className="relative">
        <input
          ref={inputRef}
          id={inputId}
          type="text"
          role="combobox"
          aria-autocomplete="list"
          aria-expanded={open}
          aria-controls={listboxId}
          aria-activedescendant={activeId}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onBlur={(event) => {
            onBlur?.(event);
            close();
          }}
          className={cn(
            "min-h-12 w-full border-2 bg-background px-4 text-foreground placeholder:text-muted",
            error ? "border-danger" : "border-border",
          )}
          {...inputProps}
        />

        <div
          hidden={!open}
          className="absolute inset-x-0 top-full z-30 mt-1 overflow-hidden rounded-xl border border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)]"
        >
          <ul
            id={listboxId}
            role="listbox"
            aria-label={`Suggestions : ${label}`}
            className="max-h-72 overflow-y-auto py-1"
          >
            {suggestions.map((suggestion, index) => (
              <li
                key={suggestion.id}
                id={`${listboxId}-${index}`}
                role="option"
                aria-selected={index === activeIndex}
                // mousedown : choisir avant que le champ ne perde le focus (et ne ferme la liste).
                onMouseDown={(event) => {
                  event.preventDefault();
                  choose(suggestion);
                }}
                onMouseMove={() => setActiveIndex(index)}
                className={cn(
                  "flex min-h-11 cursor-pointer flex-col justify-center px-4 py-2",
                  index === activeIndex && "bg-surface-raised",
                )}
              >
                <span className="font-medium">{suggestion.label}</span>
                {suggestion.context && (
                  <span className="text-sm text-muted">{suggestion.context}</span>
                )}
              </li>
            ))}
          </ul>
          {/* Attribution exigée par la licence ODbL des données OpenStreetMap. */}
          <p className="border-t border-border px-4 py-1.5 text-xs text-muted">
            Données ©{" "}
            <a
              href="https://www.openstreetmap.org/copyright"
              target="_blank"
              rel="noopener noreferrer"
              tabIndex={-1}
              onMouseDown={(event) => event.preventDefault()}
              className="underline"
            >
              contributeurs OpenStreetMap
            </a>
          </p>
        </div>
      </div>
      {hint && (
        <p id={hintId} className="mt-2 text-sm text-muted">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} className="mt-2 text-sm text-danger">
          {error}
        </p>
      )}
      <p role="status" className="sr-only">
        {status}
      </p>
    </div>
  );
}
