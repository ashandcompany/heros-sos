import { redirect } from "next/navigation";
import { routes } from "@/lib/routes";

/** `/heros` n'a pas de contenu propre : on ouvre le tableau de bord. */
export default function HeroSpacePage() {
  redirect(routes.heroDashboard);
}
