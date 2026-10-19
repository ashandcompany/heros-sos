"use client";

import {
  Children,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { cn } from "./cn";

interface CarouselProps {
  /** Nom accessible du carrousel. */
  label: string;
  previousLabel?: string;
  nextLabel?: string;
  /** Chaque enfant devient une diapositive. */
  children: ReactNode;
}

function ChevronIcon({ direction }: { direction: "left" | "right" }) {
  return (
    <svg viewBox="0 0 20 20" aria-hidden="true" className="size-5 fill-none stroke-current">
      <path
        d={direction === "left" ? "M12.5 15 7.5 10l5-5" : "m7.5 15 5-5-5-5"}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

const controlClasses =
  "inline-flex size-11 items-center justify-center rounded-md border border-[#9a8a6c] bg-paper text-background shadow-[inset_0_1px_0_rgb(255_255_255/0.6),inset_0_-2px_0_rgb(0_0_0/0.15),0_3px_0_#0d0906] hover:bg-accent active:translate-y-0.5 active:shadow-[0_1px_0_#0d0906] aria-disabled:cursor-not-allowed aria-disabled:opacity-40 aria-disabled:hover:bg-paper";

/**
 * Carrousel accessible fait maison.
 * - Swipe natif sur mobile grâce au défilement horizontal + scroll-snap.
 * - Boutons précédent/suivant, pastilles de position, flèches clavier / Début / Fin.
 * - Respecte `prefers-reduced-motion` (défilement instantané).
 */
export function Carousel({
  label,
  previousLabel = "Précédent",
  nextLabel = "Suivant",
  children,
}: CarouselProps) {
  const slides = Children.toArray(children);
  const count = slides.length;
  const trackId = useId();
  const trackRef = useRef<HTMLDivElement>(null);
  const frameRef = useRef<number | null>(null);

  const [activeIndex, setActiveIndex] = useState(0);
  const [visibleCount, setVisibleCount] = useState(1);
  const [atStart, setAtStart] = useState(true);
  const [atEnd, setAtEnd] = useState(count <= 1);

  const positions = Math.max(count - visibleCount + 1, 1);

  const measure = useCallback(() => {
    const track = trackRef.current;
    const firstSlide = track?.firstElementChild as HTMLElement | null;
    if (!track || !firstSlide) return;

    const { scrollLeft, scrollWidth, clientWidth } = track;
    const slideElements = Array.from(track.children) as HTMLElement[];

    let closestIndex = 0;
    let closestDistance = Infinity;
    slideElements.forEach((slide, index) => {
      const distance = Math.abs(slide.offsetLeft - scrollLeft);
      if (distance < closestDistance) {
        closestDistance = distance;
        closestIndex = index;
      }
    });

    setActiveIndex(closestIndex);
    setVisibleCount(Math.max(Math.round(clientWidth / firstSlide.offsetWidth), 1));
    setAtStart(scrollLeft <= 1);
    setAtEnd(scrollLeft + clientWidth >= scrollWidth - 1);
  }, []);

  const scheduleMeasure = useCallback(() => {
    if (frameRef.current !== null) cancelAnimationFrame(frameRef.current);
    frameRef.current = requestAnimationFrame(measure);
  }, [measure]);

  useEffect(() => {
    const track = trackRef.current;
    if (!track) return;

    const observer = new ResizeObserver(scheduleMeasure);
    observer.observe(track);

    return () => {
      observer.disconnect();
      if (frameRef.current !== null) cancelAnimationFrame(frameRef.current);
    };
  }, [scheduleMeasure]);

  const goTo = (index: number) => {
    const track = trackRef.current;
    if (!track) return;

    const target = track.children[Math.min(Math.max(index, 0), count - 1)] as
      HTMLElement | undefined;
    if (!target) return;

    const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    track.scrollTo({ left: target.offsetLeft, behavior: prefersReducedMotion ? "auto" : "smooth" });
  };

  const goToPrevious = () => {
    if (!atStart) goTo(activeIndex - 1);
  };

  const goToNext = () => {
    if (!atEnd) goTo(activeIndex + 1);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const actions: Record<string, () => void> = {
      ArrowLeft: goToPrevious,
      ArrowRight: goToNext,
      Home: () => goTo(0),
      End: () => goTo(count - 1),
    };
    const action = actions[event.key];
    if (action) {
      event.preventDefault();
      action();
    }
  };

  const firstVisible = activeIndex + 1;
  const lastVisible = Math.min(activeIndex + visibleCount, count);
  const status =
    firstVisible === lastVisible
      ? `Élément ${firstVisible} sur ${count}`
      : `Éléments ${firstVisible} à ${lastVisible} sur ${count}`;

  if (count === 0) return null;

  return (
    <div role="region" aria-roledescription="carrousel" aria-label={label}>
      <div
        id={trackId}
        ref={trackRef}
        tabIndex={0}
        onScroll={scheduleMeasure}
        onKeyDown={handleKeyDown}
        aria-label="Liste défilante, utilisez les flèches gauche et droite pour naviguer"
        className="relative -mx-1 flex snap-x snap-mandatory scrollbar-none gap-4 overflow-x-auto overscroll-x-contain px-1 py-2"
      >
        {slides.map((slide, index) => (
          <div
            key={index}
            role="group"
            aria-roledescription="diapositive"
            aria-label={`${index + 1} sur ${count}`}
            className="w-[85%] shrink-0 snap-start sm:w-[calc((100%-1rem)/2)] lg:w-[calc((100%-2rem)/3)]"
          >
            {slide}
          </div>
        ))}
      </div>

      <div className="mt-6 flex items-center justify-between gap-4">
        <div className="flex flex-wrap gap-1" role="group" aria-label="Positions">
          {Array.from({ length: positions }, (_, index) => {
            const isCurrent = index === Math.min(activeIndex, positions - 1);
            return (
              <button
                key={index}
                type="button"
                onClick={() => goTo(index)}
                aria-controls={trackId}
                aria-current={isCurrent ? "true" : undefined}
                aria-label={`Afficher l'élément ${index + 1}`}
                className="group inline-flex size-6 items-center justify-center"
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    "h-2",
                    isCurrent ? "w-6 bg-terminal" : "w-2 bg-muted/50 group-hover:bg-muted",
                  )}
                />
              </button>
            );
          })}
        </div>

        <div className="flex gap-2">
          <button
            type="button"
            onClick={goToPrevious}
            aria-controls={trackId}
            aria-disabled={atStart}
            aria-label={previousLabel}
            className={controlClasses}
          >
            <ChevronIcon direction="left" />
          </button>
          <button
            type="button"
            onClick={goToNext}
            aria-controls={trackId}
            aria-disabled={atEnd}
            aria-label={nextLabel}
            className={controlClasses}
          >
            <ChevronIcon direction="right" />
          </button>
        </div>
      </div>

      <p className="sr-only" aria-live="polite" aria-atomic="true">
        {status}
      </p>
    </div>
  );
}
