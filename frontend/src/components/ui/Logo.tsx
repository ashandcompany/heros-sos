import { cn } from "./cn";

/** Logo placeholder : emblème « signal » + logotype texte. */
export function Logo({ className }: { className?: string }) {
  return (
    <span className={cn("inline-flex items-center gap-2", className)}>
      <svg viewBox="0 0 32 32" aria-hidden="true" className="size-8 shrink-0">
        <path
          d="M16 2 4 7v8.5C4 23 9.2 28.6 16 30c6.8-1.4 12-7 12-14.5V7L16 2Z"
          className="fill-surface-raised stroke-primary"
          strokeWidth="2"
          strokeLinejoin="round"
        />
        <path
          d="M8 17h4l2-5 4 10 2-5h4"
          className="stroke-accent"
          fill="none"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
      <span className="font-display text-lg font-bold tracking-tight">
        Heros<span className="text-primary">-SOS</span>
      </span>
    </span>
  );
}
