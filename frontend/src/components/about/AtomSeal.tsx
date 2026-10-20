import { cn } from "@/components/ui/cn";

const orbitAngles = [0, 60, 120];

/**
 * Sceau du Bureau : atome à trois orbites dans un médaillon à texte circulaire. Décoratif.
 * Les orbites tournent lentement, sauf sous prefers-reduced-motion.
 */
export function AtomSeal({ className }: { className?: string }) {
  return (
    <svg aria-hidden="true" viewBox="0 0 320 320" className={cn("h-auto", className)}>
      <defs>
        {/* Cercle parcouru par le texte, départ en haut à gauche */}
        <path
          id="atom-seal-text"
          d="M 160 160 m -128 0 a 128 128 0 1 1 256 0 a 128 128 0 1 1 -256 0"
        />
      </defs>

      <circle cx="160" cy="160" r="156" className="fill-paper stroke-edge" strokeWidth="4" />
      <circle cx="160" cy="160" r="112" className="fill-primary stroke-ink" strokeWidth="4" />
      <circle
        cx="160"
        cy="160"
        r="146"
        className="fill-none stroke-ink"
        strokeWidth="1.5"
        strokeDasharray="2 6"
      />

      <text className="fill-ink font-mono text-[15px] font-semibold tracking-[0.2em] uppercase">
        <textPath href="#atom-seal-text" startOffset="0">
          Bureau des interventions extraordinaires ★ Fondé en 1957 ★
        </textPath>
      </text>

      <g className="origin-center [transform-box:fill-box] motion-safe:animate-[spin_60s_linear_infinite]">
        {orbitAngles.map((angle) => (
          <g key={angle} transform={`rotate(${angle} 160 160)`}>
            <ellipse
              cx="160"
              cy="160"
              rx="88"
              ry="30"
              className="fill-none stroke-paper"
              strokeWidth="5"
            />
            <circle cx="248" cy="160" r="8" className="fill-mustard stroke-ink" strokeWidth="3" />
          </g>
        ))}
      </g>

      <circle cx="160" cy="160" r="20" className="fill-accent stroke-paper" strokeWidth="5" />
    </svg>
  );
}
