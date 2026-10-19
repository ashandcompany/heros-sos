"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/components/ui/cn";
import { citizenNav } from "@/lib/routes";

/** Sous-navigation de l'espace personnel (défilement horizontal sur mobile). */
export function CitizenNav() {
  const pathname = usePathname();

  return (
    <nav aria-label="Espace personnel" className="-mx-4 mb-8 overflow-x-auto px-4 sm:mx-0 sm:px-0">
      <ul className="flex w-max gap-2 border-b border-border sm:w-auto">
        {citizenNav.map((item) => {
          const isCurrent = pathname === item.href;
          return (
            <li key={item.href}>
              <Link
                href={item.href}
                aria-current={isCurrent ? "page" : undefined}
                className={cn(
                  "-mb-px inline-flex min-h-11 items-center border-b-2 px-3 text-sm font-medium whitespace-nowrap",
                  isCurrent
                    ? "border-primary text-foreground"
                    : "border-transparent text-muted hover:text-foreground",
                )}
              >
                {item.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
