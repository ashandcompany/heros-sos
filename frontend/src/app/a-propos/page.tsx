import type { Metadata } from "next";
import { AboutHero } from "@/components/about/AboutHero";
import { JoinSection } from "@/components/about/JoinSection";
import { MissionSection } from "@/components/about/MissionSection";
import { OrgChartSection } from "@/components/about/OrgChartSection";
import { PrinciplesSection } from "@/components/about/PrinciplesSection";
import { TimelineSection } from "@/components/about/TimelineSection";

export const metadata: Metadata = {
  title: "À propos",
  description:
    "Depuis 1957, le Bureau relie les citoyens aux super-héros : mission, règlement, histoire et organisation de Heros-SOS.",
};

export default function AboutPage() {
  return (
    <>
      <AboutHero />
      <MissionSection />
      <PrinciplesSection />
      <TimelineSection />
      <OrgChartSection />
      <JoinSection />
    </>
  );
}
