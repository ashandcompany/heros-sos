import { HeroSection } from "@/components/home/HeroSection";
import { HowItWorksSection } from "@/components/home/HowItWorksSection";
import { LatestIncidentsSection } from "@/components/home/LatestIncidentsSection";
import { SuperHeroesSection } from "@/components/home/SuperHeroesSection";

export default function HomePage() {
  return (
    <>
      <HeroSection />
      <LatestIncidentsSection />
      <HowItWorksSection />
      <SuperHeroesSection />
    </>
  );
}
