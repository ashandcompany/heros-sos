import { cn } from "./cn";

const S_PATH = "M6 0h18v7H8v5.5h10l6 6V26l-6 6H0v-7h16v-5.5H6l-6-6V6Z";

/**
 * Monogramme « SOS » : deux S chanfreinés massifs, O bauhaus posé par-dessus
 * (détouré par un filet de la couleur du fond) avec un voyant au centre.
 */
function Monogram({ className }: { className?: string }) {
  return (
    <svg viewBox="-7 -7 86 46" role="img" aria-label="Heros-SOS" className={className}>
      <rect x="-7" y="-7" width="86" height="46" className="fill-primary" />
      <path d={S_PATH} className="fill-paper" />
      <path d={S_PATH} transform="translate(48 0)" className="fill-paper" />
      <circle cx="36" cy="16" r="17.5" className="fill-primary" />
      <path
        fillRule="evenodd"
        d="M36 0a16 16 0 1 1 0 32 16 16 0 0 1 0-32Zm0 8.5a7.5 7.5 0 1 0 0 15 7.5 7.5 0 0 0 0-15Z"
        className="fill-paper"
      />
      <circle cx="36" cy="16" r="3" className="fill-accent" />
    </svg>
  );
}

/** Logo : monogramme dans un cartouche, suivi du nom complet et de la mention de service. */
export function Logo({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-stretch border-[3px] border-edge bg-background shadow-[3px_3px_0_0_var(--color-shadow)]",
        className,
      )}
    >
      <Monogram className="h-10 w-auto shrink-0" />
      <span
        aria-hidden="true"
        className="hidden flex-col justify-center border-l-[3px] border-edge px-2.5 leading-none whitespace-nowrap sm:flex"
      >
        <span className="font-display text-sm font-black tracking-[0.3em] uppercase">
          Heros-SOS
        </span>
        <span className="mt-1 font-mono text-[0.55rem] tracking-[0.2em] text-muted uppercase">
          Service d&apos;intervention
        </span>
      </span>
    </span>
  );
}
