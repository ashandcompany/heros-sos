import type { CSSProperties } from "react";
import { cn } from "./cn";

interface TypewriterProps {
  text: string;
  className?: string;
}

/**
 * Texte révélé caractère par caractère (animation CSS, désactivée sous prefers-reduced-motion).
 * Le texte complet reste lisible par les lecteurs d'écran ; l'effet est purement visuel.
 */
export function Typewriter({ text, className }: TypewriterProps) {
  let index = 0;

  return (
    <>
      <span className="sr-only">{text}</span>
      <span aria-hidden="true" className={className}>
        {text.split(" ").map((word, wordIndex) => (
          <span key={wordIndex} className="inline-block whitespace-nowrap">
            {Array.from(word).map((char) => (
              <span
                key={index}
                className={cn("tw-char")}
                style={{ "--i": index++ } as CSSProperties}
              >
                {char}
              </span>
            ))}
            {" "}
          </span>
        ))}
      </span>
    </>
  );
}
