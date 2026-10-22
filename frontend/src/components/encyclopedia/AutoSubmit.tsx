"use client";

import { useSearchParams } from "next/navigation";
import { useEffect, useRef } from "react";

const SEARCH_DELAY_MS = 350;

const isSearchInput = (target: EventTarget | null): target is HTMLInputElement =>
  target instanceof HTMLInputElement && target.type === "search";

/**
 * Applique les filtres du formulaire parent dès qu'ils changent (recherche après une courte pause
 * de frappe). Sans JavaScript, le bouton du formulaire suffit. Lit l'URL : à rendre sous <Suspense>.
 */
export function AutoSubmit() {
  const marker = useRef<HTMLSpanElement>(null);
  const searchParams = useSearchParams();

  useEffect(() => {
    const form = marker.current?.closest("form");
    if (!form) return;
    let timer: ReturnType<typeof setTimeout> | undefined;

    const submit = () => {
      clearTimeout(timer);
      form.requestSubmit();
    };
    const onInput = (event: Event) => {
      if (!isSearchInput(event.target)) return;
      clearTimeout(timer);
      timer = setTimeout(submit, SEARCH_DELAY_MS);
    };
    const onChange = (event: Event) => {
      // La recherche est déjà soumise pendant la frappe : son `change` (à la sortie du champ) ferait doublon.
      if (!isSearchInput(event.target)) submit();
    };

    // Champs vides (« Toutes », tri par défaut) retirés de l'URL : `?onglet=vilains&menace=eleve`
    // plutôt que `?onglet=vilains&q=&menace=eleve&statut=…`.
    const onFormData = ({ formData }: FormDataEvent) => {
      for (const [name, value] of [...formData.entries()]) {
        if (value === "") formData.delete(name);
      }
    };

    form.addEventListener("input", onInput);
    form.addEventListener("change", onChange);
    form.addEventListener("formdata", onFormData);
    return () => {
      clearTimeout(timer);
      form.removeEventListener("input", onInput);
      form.removeEventListener("change", onChange);
      form.removeEventListener("formdata", onFormData);
    };
  }, []);

  // Les champs ne sont pas contrôlés : quand l'URL change ailleurs (filtre retiré, retour arrière),
  // on y recopie ses valeurs, sauf dans le champ en cours de saisie.
  useEffect(() => {
    const form = marker.current?.closest("form");
    if (!form) return;
    for (const field of form.querySelectorAll<HTMLInputElement | HTMLSelectElement>(
      "input[name]:not([type=hidden]), select[name]",
    )) {
      if (field !== document.activeElement) field.value = searchParams.get(field.name) ?? "";
    }
  }, [searchParams]);

  return <span ref={marker} hidden />;
}
