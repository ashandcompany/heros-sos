import { ButtonLink } from "@/components/ui/ButtonLink";

/**
 * Codes de secours, montrés cette seule fois : l'API n'en garde que l'empreinte. Chacun remplace un code de
 * l'application, une fois, si l'appareil est perdu.
 */
export function RecoveryCodesNotice({
  codes,
  title,
  continueHref,
  continueLabel = "Continuer",
}: {
  codes: string[];
  title: string;
  /** Page où poursuivre : sans lien, les codes restent affichés (régénération depuis la page de sécurité). */
  continueHref?: string;
  continueLabel?: string;
}) {
  return (
    <div className="space-y-5">
      <p role="status" className="rounded-xl border border-success/40 bg-success/10 p-4 text-sm">
        <span className="block font-semibold text-success">{title}</span>
        Les codes de secours ci-dessous ne seront plus jamais affichés.
      </p>
      <p className="text-sm text-muted">
        Si vous perdez votre téléphone, chacun de ces codes vous permet de vous connecter, une seule
        fois. Notez-les ou imprimez-les, et gardez-les hors de votre ordinateur et loin de votre mot
        de passe.
      </p>
      <ul
        aria-label="Codes de secours"
        className="grid gap-2 border-2 border-border bg-background p-4 font-mono text-sm sm:grid-cols-2"
      >
        {codes.map((code) => (
          <li key={code}>
            <code className="select-all">{code}</code>
          </li>
        ))}
      </ul>
      {continueHref && (
        <ButtonLink href={continueHref} size="lg" className="w-full">
          {continueLabel}
        </ButtonLink>
      )}
    </div>
  );
}
