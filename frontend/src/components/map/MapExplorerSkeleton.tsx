/** Emplacement de `IncidentMapExplorer` pendant le chargement des données. */
export function MapExplorerSkeleton() {
  return (
    <div aria-hidden="true" className="grid gap-6 lg:grid-cols-[minmax(20rem,24rem)_1fr]">
      <div className="h-[60vh] min-h-80 border-2 border-border bg-surface motion-safe:animate-pulse lg:order-2 lg:h-[min(calc(100vh-11rem),720px)]" />
      <div className="hidden border-2 border-border bg-surface motion-safe:animate-pulse lg:order-1 lg:block" />
    </div>
  );
}
