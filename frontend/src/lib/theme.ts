export type Theme = "dark" | "light";

/** Clé `localStorage` du thème choisi. Absente : thème sombre (thème par défaut du site). */
export const THEME_STORAGE_KEY = "theme";

/** Couleur de la barre du navigateur mobile (`<meta name="theme-color">`) pour chaque thème. */
export const themeColors: Record<Theme, string> = {
  dark: "#111315",
  light: "#efe9da",
};

/**
 * Script exécuté dans <head> avant le premier affichage : applique le thème enregistré
 * pour éviter un flash du thème sombre. Doit rester autonome (aucun import).
 */
export const themeInitScript = `try{if(localStorage.getItem(${JSON.stringify(
  THEME_STORAGE_KEY,
)})==="light")document.documentElement.dataset.theme="light"}catch(e){}`;
