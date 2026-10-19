export function SkipLink() {
  return (
    <a
      href="#contenu"
      className="sr-only rounded-full bg-primary px-4 py-2 font-semibold text-primary-foreground focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-50"
    >
      Aller au contenu principal
    </a>
  );
}
