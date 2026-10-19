export interface Country {
  /** Code ISO 3166-1 alpha-2. */
  code: string;
  name: string;
}

export const DEFAULT_COUNTRY = "FR";

/** Pays proposés à l'inscription. À remplacer par la liste fournie par l'API si besoin. */
export const countries: Country[] = [
  { code: "DE", name: "Allemagne" },
  { code: "BE", name: "Belgique" },
  { code: "CA", name: "Canada" },
  { code: "ES", name: "Espagne" },
  { code: "US", name: "États-Unis" },
  { code: "FR", name: "France" },
  { code: "IT", name: "Italie" },
  { code: "LU", name: "Luxembourg" },
  { code: "MC", name: "Monaco" },
  { code: "NL", name: "Pays-Bas" },
  { code: "PT", name: "Portugal" },
  { code: "GB", name: "Royaume-Uni" },
  { code: "CH", name: "Suisse" },
];
