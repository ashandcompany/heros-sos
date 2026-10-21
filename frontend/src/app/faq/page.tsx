import type { Metadata } from "next";
import Link from "next/link";
import { FaqCategory, type FaqEntry } from "@/components/faq/FaqCategory";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Questions fréquentes",
  description:
    "Signaler un incident, devenir super-héros, gérer son compte : les réponses aux questions les plus posées au Bureau.",
};

interface Category {
  id: string;
  code: string;
  title: string;
  entries: FaqEntry[];
}

const categories: Category[] = [
  {
    id: "signaler",
    code: "A",
    title: "Signaler un incident",
    entries: [
      {
        question: "Comment signaler un incident ?",
        answer: (
          <p>
            Connectez-vous, puis rendez-vous dans{" "}
            <Link href={routes.profileIncidents}>Mes incidents</Link>. Indiquez si vous êtes témoin
            ou victime, le type d&apos;incident, le lieu et, si vous le connaissez, le super-vilain
            impliqué. Comptez moins d&apos;une minute.
          </p>
        ),
      },
      {
        question: "Faut-il un compte pour signaler ?",
        answer: (
          <p>
            Oui. Le compte permet au Bureau de vérifier les déclarations, de vous joindre pendant
            l&apos;intervention et de vous tenir informé. L&apos;
            <Link href={routes.signup}>inscription</Link> est gratuite.
          </p>
        ),
      },
      {
        question: "Que se passe-t-il après ma déclaration ?",
        answer: (
          <>
            <p>
              Le Bureau vérifie la déclaration, puis un super-héros disponible près du lieu prend le
              dossier en charge. Vous suivez l&apos;avancement depuis votre espace personnel.
            </p>
            <p>
              Une fois l&apos;intervention terminée, vous confirmez son issue et pouvez noter
              l&apos;agent.
            </p>
          </>
        ),
      },
      {
        question: "C'est une urgence, que faire ?",
        answer: (
          <p>
            Mettez-vous d&apos;abord à l&apos;abri. Ne tentez jamais d&apos;arrêter un super-vilain
            vous-même. Déclarez ensuite l&apos;incident : c&apos;est le seul moyen de faire
            intervenir un agent.
          </p>
        ),
      },
    ],
  },
  {
    id: "super-heros",
    code: "B",
    title: "Super-héros",
    entries: [
      {
        question: "Comment devenir super-héros sur Heros-SOS ?",
        answer: (
          <p>
            Créez d&apos;abord un compte, comme tout citoyen, puis adressez votre demande de statut
            de super-héros au Bureau via la <Link href={routes.contact}>page de contact</Link>. Le
            Bureau l&apos;examine : votre espace professionnel s&apos;ouvre une fois la demande
            validée.
          </p>
        ),
      },
      {
        question: "Comment les super-héros sont-ils notés ?",
        answer: (
          <p>
            Après chaque intervention, le citoyen qui a signalé l&apos;incident peut attribuer une
            note de 1 à 5 et laisser un commentaire, vérifié par la modération. Les meilleurs agents
            apparaissent dans l&apos;<Link href={routes.encyclopedia}>encyclopédie</Link>.
          </p>
        ),
      },
      {
        question: "Puis-je choisir le super-héros qui intervient ?",
        answer: (
          <p>
            Non. Pour aller au plus vite, le dossier revient à l&apos;agent disponible le plus
            proche des faits.
          </p>
        ),
      },
    ],
  },
  {
    id: "compte",
    code: "C",
    title: "Mon compte",
    entries: [
      {
        question: "Je n'ai pas reçu l'email d'activation.",
        answer: (
          <p>
            Vérifiez vos courriers indésirables. Le lien d&apos;activation est valable 24 heures :
            passé ce délai, demandez-en un nouveau depuis la page d&apos;activation.
          </p>
        ),
      },
      {
        question: "J'ai oublié mon mot de passe.",
        answer: (
          <p>
            Utilisez la page <Link href={routes.forgotPassword}>Mot de passe oublié</Link>. Vous
            recevrez un lien de réinitialisation valable 1 heure.
          </p>
        ),
      },
      {
        question: "Comment modifier mes informations ?",
        answer: (
          <p>
            Depuis <Link href={routes.profile}>votre espace personnel</Link> : téléphone, adresse,
            photo, adresse email et abonnement à la newsletter.
          </p>
        ),
      },
    ],
  },
  {
    id: "donnees",
    code: "D",
    title: "Données personnelles",
    entries: [
      {
        question: "Ma déclaration est-elle publique ?",
        answer: (
          <p>
            Les incidents apparaissent sur la <Link href={routes.map}>carte publique</Link>, mais
            jamais votre nom ni vos coordonnées. Seuls le Bureau et le super-héros chargé du dossier
            y ont accès.
          </p>
        ),
      },
      {
        question: "Comment supprimer mon compte ou récupérer mes données ?",
        answer: (
          <p>
            Écrivez-nous via la <Link href={routes.contact}>page de contact</Link> en choisissant
            l&apos;objet « Mes données personnelles ». Le détail de vos droits figure dans la{" "}
            <Link href={routes.privacy}>politique de confidentialité</Link>.
          </p>
        ),
      },
      {
        question: "Le site utilise-t-il des cookies publicitaires ?",
        answer: (
          <p>
            Non. Heros-SOS n&apos;utilise ni publicité, ni mesure d&apos;audience, ni traceur tiers
            : seulement le cookie qui vous garde connecté et la mémorisation de vos préférences.
          </p>
        ),
      },
    ],
  },
];

export default function FaqPage() {
  return (
    <Container className="py-10 sm:py-14">
      <header className="max-w-3xl">
        <p className="mb-5 inline-flex flex-wrap items-center gap-x-3 gap-y-1 border-2 border-edge bg-background px-3 py-1.5 font-mono text-xs tracking-widest text-muted uppercase">
          <span>Bulletin d&apos;information</span>
          <span aria-hidden="true" className="text-border">
            |
          </span>
          <span>Mis à jour par le Bureau</span>
        </p>
        <h1 className="font-display text-4xl leading-[1.05] font-black tracking-tight text-balance uppercase sm:text-5xl">
          Questions fréquentes
        </h1>
        <p className="mt-4 text-lg text-pretty text-muted">
          Les questions les plus posées au guichet, classées par dossier. Cliquez sur une question
          pour lire la réponse.
        </p>
        <div aria-hidden="true" className="bar-tricolor mt-6 w-48" />
      </header>

      <div className="mt-12 grid items-start gap-12 lg:grid-cols-[14rem_1fr]">
        <nav aria-labelledby="faq-toc-title" className="lg:sticky lg:top-28">
          <h2
            id="faq-toc-title"
            className="font-mono text-xs font-medium tracking-widest text-accent uppercase"
          >
            Dossiers
          </h2>
          <ul className="mt-3 flex flex-wrap gap-2 lg:flex-col">
            {categories.map((category) => (
              <li key={category.id}>
                <a
                  href={`#${category.id}`}
                  className="flex min-h-11 items-center gap-3 border-2 border-border px-3 font-medium hover:border-edge hover:bg-surface-raised"
                >
                  <span aria-hidden="true" className="font-mono text-sm text-accent">
                    {category.code}
                  </span>
                  {category.title}
                </a>
              </li>
            ))}
          </ul>
        </nav>

        <div className="space-y-14">
          {categories.map((category) => (
            <FaqCategory key={category.id} {...category} />
          ))}

          <aside
            aria-labelledby="faq-contact-title"
            className="panel rivets paper relative p-6 sm:p-8"
          >
            <h2
              id="faq-contact-title"
              className="font-display text-2xl font-black tracking-wider uppercase"
            >
              Votre question n&apos;y figure pas ?
            </h2>
            <p className="mt-2 text-pretty text-ink/85">
              Le guichet de la correspondance vous répond par email sous 5 jours ouvrés.
            </p>
            <ButtonLink href={routes.contact} className="mt-5">
              Écrire au Bureau
            </ButtonLink>
          </aside>
        </div>
      </div>
    </Container>
  );
}
