/** Numéro administratif décoratif dérivé d'un identifiant (« incident-12 » → « HSOS-0012 »). */
export function dossierNumber(id: string, prefix = "HSOS"): string {
  const digits = id.match(/\d+/)?.[0] ?? "0";
  return `${prefix}-${digits.padStart(4, "0")}`;
}
