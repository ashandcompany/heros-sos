import Image from "next/image";
import { cn } from "./cn";

interface WallPosterProps {
  src: string;
  width: number;
  height: number;
  caption: string;
  className?: string;
}

/** Affiche punaisée au mur : décorative, masquée aux lecteurs d'écran et sur petit écran. */
export function WallPoster({ src, width, height, caption, className }: WallPosterProps) {
  return (
    <figure
      aria-hidden="true"
      className={cn(
        "relative hidden w-56 shrink-0 border-4 border-edge bg-paper p-2 shadow-[8px_8px_0_0_var(--color-shadow)] lg:block xl:w-64",
        className,
      )}
    >
      <span className="led absolute -top-2 left-1/2 -translate-x-1/2 bg-brick shadow-none" />
      <Image
        src={src}
        alt=""
        width={width}
        height={height}
        sizes="256px"
        className="h-auto w-full"
      />
      <figcaption className="mt-1.5 font-mono text-[0.6rem] tracking-widest text-ink/75 uppercase">
        {caption}
      </figcaption>
    </figure>
  );
}
