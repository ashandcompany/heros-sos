import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";

const memoFields = [
  { label: "De", value: "La Direction du Bureau" },
  { label: "À", value: "L'ensemble des citoyens" },
  { label: "Objet", value: "Notre mission" },
  { label: "Réf.", value: "7-B / 57 / 001" },
];

const readouts = [
  { value: "1957", label: "Année de fondation" },
  { value: "24/24", label: "Guichet ouvert" },
  { value: "7-B", label: "Formulaire unique" },
  { value: "03", label: "Guichets par dossier" },
];

/** Note de service tapée à la machine et écran de contrôle aux chiffres du Bureau. */
export function MissionSection() {
  return (
    <section aria-labelledby="mission-title" className="py-14 sm:py-20">
      <Container>
        <SectionHeading
          id="mission-title"
          section="01"
          eyebrow="Note de service"
          title="Une mission, un guichet"
          className="mb-12"
        />

        <div className="grid items-start gap-12 lg:grid-cols-[1.4fr_1fr]">
          <article
            aria-labelledby="memo-title"
            className="panel rivets paper relative p-6 pt-8 sm:p-10"
          >
            <h3 id="memo-title" className="sr-only">
              Note de service : notre mission
            </h3>
            <dl className="grid gap-y-2 font-mono text-sm sm:grid-cols-[6rem_1fr]">
              {memoFields.map((field) => (
                <div key={field.label} className="contents">
                  <dt className="font-semibold tracking-widest uppercase">{field.label} :</dt>
                  <dd className="field-line mb-2 pb-1 sm:mb-0">{field.value}</dd>
                </div>
              ))}
            </dl>

            <div aria-hidden="true" className="rule-double mt-6" />

            <div className="mt-6 space-y-4 text-pretty text-ink/90">
              <p>
                Quand un super-vilain frappe, chaque minute compte. Le Bureau a été créé pour
                qu&apos;aucun citoyen ne reste seul face à un rayon congelant, une horde de robots
                ou un pont qui rouille à vue d&apos;œil.
              </p>
              <p>
                Vous déclarez l&apos;incident ; nous le vérifions ; l&apos;agent disponible le plus
                proche intervient. Une fois le calme revenu, c&apos;est vous qui notez son travail.
                Les meilleurs agents montent au registre, les autres sont convoqués.
              </p>
              <p className="font-stamp tracking-wide">
                Le Bureau vous remercie de votre vigilance.
              </p>
            </div>

            <div className="mt-8 flex items-end justify-between gap-6">
              <div className="font-mono text-xs tracking-widest uppercase">
                <div aria-hidden="true" className="field-line mb-2 h-10 w-40" />
                Visa de la Direction
              </div>
              <span aria-hidden="true" className="stamp text-xl sm:text-2xl">
                Approuvé
              </span>
            </div>
          </article>

          <div className="crt">
            <div className="crt__screen">
              <p className="mb-4 flex items-center gap-2 font-mono text-xs tracking-widest text-terminal uppercase">
                <span aria-hidden="true" className="led" />
                Tableau de contrôle
              </p>
              <dl className="grid grid-cols-2 gap-4">
                {readouts.map((readout) => (
                  <div key={readout.label} className="flex flex-col-reverse gap-2">
                    <dt className="font-mono text-[0.65rem] tracking-widest text-muted uppercase">
                      {readout.label}
                    </dt>
                    <dd className="digital self-start text-2xl sm:text-3xl">{readout.value}</dd>
                  </div>
                ))}
              </dl>
            </div>
          </div>
        </div>
      </Container>
    </section>
  );
}
