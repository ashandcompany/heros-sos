import { formatRating } from "@/lib/format";

const STAR_PATH =
  "M10 1.5l2.6 5.3 5.9.9-4.3 4.1 1 5.8L10 14.9l-5.2 2.7 1-5.8L1.5 7.7l5.9-.9L10 1.5z";

function Stars({ className }: { className: string }) {
  return (
    <span className={`flex gap-0.5 ${className}`}>
      {Array.from({ length: 5 }, (_, index) => (
        <svg key={index} viewBox="0 0 20 20" className="size-4 shrink-0 fill-current">
          <path d={STAR_PATH} />
        </svg>
      ))}
    </span>
  );
}

/** Note sur 5 : étoiles décoratives + valeur numérique, avec un libellé complet pour les lecteurs d'écran. */
export function RatingStars({ rating }: { rating: number }) {
  const percentage = Math.min(Math.max(rating / 5, 0), 1) * 100;

  return (
    <span className="inline-flex items-center gap-2">
      <span aria-hidden="true" className="relative inline-flex">
        <Stars className="text-border" />
        <span className="absolute inset-0 overflow-hidden" style={{ width: `${percentage}%` }}>
          <Stars className="text-rating" />
        </span>
      </span>
      <span aria-hidden="true" className="text-sm font-semibold">
        {formatRating(rating)}
      </span>
      <span className="sr-only">Note moyenne : {formatRating(rating)} sur 5</span>
    </span>
  );
}
