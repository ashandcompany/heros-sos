import type { Metadata } from "next";
import Link from "next/link";
import { LegalPage, ToComplete, type LegalSection } from "@/components/legal/LegalPage";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Mentions légales",
  description: "Éditeur, directeur de la publication et hébergeur du site Heros-SOS.",
};

// Informations exigées par l'article 6 de la loi n° 2004-575 du 21 juin 2004 (LCEN).
const sections: LegalSection[] = [
  {
    id: "editeur",
    title: "Éditeur du site",
    content: (
      <>
        <p>
          Le site Heros-SOS est édité par{" "}
          <ToComplete>
            nom de la personne ou raison sociale, forme juridique et capital s&apos;il s&apos;agit
            d&apos;une société
          </ToComplete>
          .
        </p>
        <ul>
          <li>
            Adresse : <ToComplete>adresse postale du siège ou du domicile</ToComplete>
          </li>
          <li>
            Immatriculation : <ToComplete>numéro RCS ou SIREN, si applicable</ToComplete>
          </li>
          <li>
            TVA intracommunautaire : <ToComplete>numéro, si applicable</ToComplete>
          </li>
          <li>
            Téléphone : <ToComplete>numéro de téléphone</ToComplete>
          </li>
          <li>
            Email : <ToComplete>adresse email de contact</ToComplete>
          </li>
        </ul>
      </>
    ),
  },
  {
    id: "directeur-publication",
    title: "Directeur de la publication",
    content: (
      <p>
        Le directeur de la publication est{" "}
        <ToComplete>prénom et nom, et qualité (gérant, président…)</ToComplete>.
      </p>
    ),
  },
  {
    id: "hebergement",
    title: "Hébergement",
    content: (
      <>
        <p>
          Le site est hébergé par <ToComplete>nom ou raison sociale de l&apos;hébergeur</ToComplete>
          .
        </p>
        <ul>
          <li>
            Adresse : <ToComplete>adresse postale de l&apos;hébergeur</ToComplete>
          </li>
          <li>
            Téléphone : <ToComplete>numéro de téléphone de l&apos;hébergeur</ToComplete>
          </li>
        </ul>
      </>
    ),
  },
  {
    id: "contact",
    title: "Nous contacter",
    content: (
      <p>
        Pour toute question sur le site, vous pouvez écrire à{" "}
        <ToComplete>adresse email de contact</ToComplete> ou passer par la{" "}
        <Link href={routes.contact}>page de contact</Link>.
      </p>
    ),
  },
  {
    id: "propriete-intellectuelle",
    title: "Propriété intellectuelle",
    content: (
      <>
        <p>
          Les textes, illustrations, logos et la charte graphique de Heros-SOS sont protégés par le
          droit d&apos;auteur. Toute reproduction ou réutilisation, totale ou partielle, sans
          autorisation écrite de l&apos;éditeur est interdite.
        </p>
        <p>
          Les contenus publiés par les utilisateurs (déclarations d&apos;incidents, commentaires,
          vidéos) restent sous la responsabilité de leurs auteurs.
        </p>
      </>
    ),
  },
  {
    id: "donnees-personnelles",
    title: "Données personnelles et cookies",
    content: (
      <p>
        La manière dont nous collectons et utilisons vos données, les cookies déposés et vos droits
        sont décrits dans notre <Link href={routes.privacy}>politique de confidentialité</Link>.
      </p>
    ),
  },
  {
    id: "credits",
    title: "Crédits",
    content: (
      <ul>
        <li>
          Fonds de carte : ©{" "}
          <a href="https://www.openstreetmap.org/copyright" rel="noopener noreferrer">
            les contributeurs d&apos;OpenStreetMap
          </a>
          , sous licence ODbL.
        </li>
        <li>
          Autocomplétion des adresses :{" "}
          <a href="https://geoservices.ign.fr/" rel="noopener noreferrer">
            Géoplateforme de l&apos;IGN
          </a>{" "}
          (Base Adresse Nationale), sous Licence Ouverte Etalab 2.0.
        </li>
        <li>
          Polices : Jost, IBM Plex Sans, IBM Plex Mono et Special Elite, sous licence SIL OFL.
        </li>
        <li>
          Illustrations : <ToComplete>auteurs et licences des images du site</ToComplete>
        </li>
      </ul>
    ),
  },
];

export default function LegalNoticePage() {
  return <LegalPage title="Mentions légales" updatedAt="2026-10-20" sections={sections} />;
}
