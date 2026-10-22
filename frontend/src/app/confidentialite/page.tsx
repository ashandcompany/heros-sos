import type { Metadata } from "next";
import Link from "next/link";
import { LegalPage, LegalTable, ToComplete, type LegalSection } from "@/components/legal/LegalPage";
import { SESSION_COOKIE, NEWSLETTER_COOKIE } from "@/lib/cookies";
import { routes } from "@/lib/routes";
import { THEME_STORAGE_KEY } from "@/lib/theme";

export const metadata: Metadata = {
  title: "Politique de confidentialité",
  description:
    "Données personnelles collectées par Heros-SOS, finalités, durées de conservation et droits.",
};

// Contenu exigé par les articles 13 et 14 du RGPD. Les noms de cookies viennent du code pour
// rester exacts si on les renomme.
const sections: LegalSection[] = [
  {
    id: "responsable",
    title: "Responsable du traitement",
    content: (
      <>
        <p>
          Le responsable du traitement de vos données est{" "}
          <ToComplete>nom ou raison sociale et adresse de l&apos;éditeur</ToComplete>.
        </p>
        <p>
          Pour toute question sur vos données, écrivez à{" "}
          <ToComplete>adresse email dédiée (ex : dpo@…)</ToComplete>.{" "}
          <ToComplete>
            coordonnées du délégué à la protection des données, s&apos;il y en a un
          </ToComplete>
        </p>
      </>
    ),
  },
  {
    id: "donnees-collectees",
    title: "Données que nous collectons",
    content: (
      <>
        <h3>Lors de la création de votre compte</h3>
        <ul>
          <li>Civilité, prénom, nom et date de naissance.</li>
          <li>Adresse email, qui sert d&apos;identifiant de connexion.</li>
          <li>Numéro de téléphone portable.</li>
          <li>Adresse postale : rue, code postal, ville et pays.</li>
          <li>
            Mot de passe, enregistré uniquement sous forme hachée (illisible, même pour nous).
          </li>
        </ul>

        <h3>Lorsque vous utilisez votre compte</h3>
        <ul>
          <li>Photo de profil, si vous en ajoutez une.</li>
          <li>
            Déclarations d&apos;incidents : votre rôle (victime ou témoin), le type d&apos;incident,
            le titre, la description, le lieu, la date et le super-vilain concerné.
          </li>
          <li>Notes et commentaires laissés sur les interventions des super-héros.</li>
          <li>
            Pour les super-héros : fiche de profil, réponses aux incidents, vidéos ajoutées et
            réputation.
          </li>
          <li>
            Pour les super-héros qui activent « Partager ma position » : leur position actuelle,
            arrondie à environ 110 m, et la date de leur consentement.
          </li>
        </ul>

        <h3>Lors de votre abonnement à la newsletter</h3>
        <ul>
          <li>Votre adresse email.</li>
        </ul>

        <h3>Automatiquement</h3>
        <ul>
          <li>
            Données techniques nécessaires à la sécurité : adresse IP, date et heure des connexions,
            tentatives de connexion échouées.
          </li>
        </ul>

        <p>
          Dans les champs de texte libre (description d&apos;un incident, commentaires), merci de ne
          pas indiquer d&apos;informations sur votre santé, vos opinions ou celles d&apos;autres
          personnes.
        </p>
      </>
    ),
  },
  {
    id: "finalites",
    title: "Pourquoi nous les utilisons",
    content: (
      <>
        <p>
          <ToComplete>faire valider ces bases légales par le porteur du projet</ToComplete>
        </p>
        <LegalTable>
          <thead>
            <tr>
              <th scope="col">Finalité</th>
              <th scope="col">Base légale</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>Créer et gérer votre compte</td>
              <td>Exécution du contrat (conditions d&apos;utilisation)</td>
            </tr>
            <tr>
              <td>
                Traiter vos déclarations d&apos;incidents, les transmettre aux super-héros et vous
                contacter pendant leur traitement
              </td>
              <td>Exécution du contrat</td>
            </tr>
            <tr>
              <td>Afficher les incidents sur la carte publique, sans vos coordonnées</td>
              <td>Intérêt légitime : informer le public des zones à risque</td>
            </tr>
            <tr>
              <td>Prévenir les super-héros proches d&apos;un nouvel incident</td>
              <td>
                Consentement, que le super-héros peut retirer à tout moment depuis « Mon profil »
              </td>
            </tr>
            <tr>
              <td>Modérer les contenus publiés</td>
              <td>Intérêt légitime : garantir un service fiable</td>
            </tr>
            <tr>
              <td>Vous envoyer la newsletter</td>
              <td>Consentement, que vous pouvez retirer à tout moment</td>
            </tr>
            <tr>
              <td>Sécuriser le site (sessions, limitation des tentatives de connexion)</td>
              <td>Intérêt légitime : protéger les comptes et le service</td>
            </tr>
          </tbody>
        </LegalTable>
      </>
    ),
  },
  {
    id: "conservation",
    title: "Combien de temps nous les gardons",
    content: (
      <LegalTable>
        <thead>
          <tr>
            <th scope="col">Données</th>
            <th scope="col">Durée de conservation</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Compte et données de profil</td>
            <td>
              Jusqu&apos;à la suppression du compte.{" "}
              <ToComplete>durée avant suppression d&apos;un compte inactif</ToComplete>
            </td>
          </tr>
          <tr>
            <td>Déclarations d&apos;incidents et interventions</td>
            <td>
              <ToComplete>durée, et sort des données après suppression du compte</ToComplete>
            </td>
          </tr>
          <tr>
            <td>Position actuelle d&apos;un super-héros</td>
            <td>
              30 minutes au plus, sans historique : chaque envoi remplace le précédent. Supprimée
              dès que le super-héros arrête le partage ou retire son consentement.
            </td>
          </tr>
          <tr>
            <td>Email d&apos;abonnement à la newsletter</td>
            <td>Jusqu&apos;à votre désabonnement</td>
          </tr>
          <tr>
            <td>Lien d&apos;activation du compte</td>
            <td>Valable 24 heures</td>
          </tr>
          <tr>
            <td>Lien de réinitialisation du mot de passe</td>
            <td>Valable 1 heure</td>
          </tr>
          <tr>
            <td>Journaux de sécurité</td>
            <td>
              <ToComplete>durée (la CNIL recommande de 6 mois à 1 an)</ToComplete>
            </td>
          </tr>
        </tbody>
      </LegalTable>
    ),
  },
  {
    id: "destinataires",
    title: "Qui y a accès",
    content: (
      <>
        <ul>
          <li>
            <strong>L&apos;équipe de modération</strong> de Heros-SOS, pour valider les comptes et
            les contenus.
          </li>
          <li>
            <strong>Les super-héros</strong> chargés d&apos;un incident, uniquement pour les
            incidents qui les concernent.
          </li>
          <li>
            <strong>Nos prestataires techniques</strong>, qui agissent sur nos instructions :
            <ul>
              <li>
                Hébergement : <ToComplete>nom et pays de l&apos;hébergeur</ToComplete>
              </li>
              <li>
                Envoi des emails : <ToComplete>nom et pays du prestataire d&apos;emails</ToComplete>
              </li>
            </ul>
          </li>
          <li>
            <strong>La Géoplateforme de l&apos;IGN</strong> (France), qui reçoit le texte des
            adresses que vous tapez pour vous proposer des suggestions. Ces requêtes passent par nos
            serveurs : votre adresse IP ne lui est pas transmise.
          </li>
          <li>
            <strong>La Fondation OpenStreetMap</strong> (Royaume-Uni), qui fournit les fonds de
            carte. Votre navigateur les télécharge directement auprès d&apos;elle, qui reçoit donc
            votre adresse IP.
          </li>
        </ul>
        <p>
          La position d&apos;un super-héros n&apos;est lue que par nos serveurs, pour choisir qui
          prévenir d&apos;un incident : elle n&apos;est affichée à personne, ni au public, ni aux
          autres super-héros, ni à la modération.
        </p>
        <p>Nous ne vendons ni ne louons vos données.</p>
      </>
    ),
  },
  {
    id: "transferts",
    title: "Transferts hors de l'Union européenne",
    content: (
      <p>
        Le Royaume-Uni, où est établie la Fondation OpenStreetMap, bénéficie d&apos;une décision
        d&apos;adéquation de la Commission européenne : vos données y sont protégées de manière
        équivalente.{" "}
        <ToComplete>
          autres transferts hors UE éventuels (hébergeur, emails) et garanties associées
        </ToComplete>
      </p>
    ),
  },
  {
    id: "cookies",
    title: "Cookies et stockage local",
    content: (
      <>
        <p>
          Heros-SOS n&apos;utilise ni cookie publicitaire, ni outil de mesure d&apos;audience, ni
          traceur tiers. Les éléments ci-dessous sont indispensables au service ou enregistrent un
          choix que vous avez fait : ils ne nécessitent pas votre consentement.
        </p>
        <LegalTable>
          <thead>
            <tr>
              <th scope="col">Nom</th>
              <th scope="col">Rôle</th>
              <th scope="col">Durée</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>
                <code>{SESSION_COOKIE}</code> (cookie)
              </td>
              <td>Vous garder connecté à votre compte</td>
              <td>
                Durée de la session. <ToComplete>durée de validité d&apos;une session</ToComplete>
              </td>
            </tr>
            <tr>
              <td>
                <code>{NEWSLETTER_COOKIE}</code> (cookie)
              </td>
              <td>Ne plus vous proposer la newsletter une fois abonné</td>
              <td>1 an</td>
            </tr>
            <tr>
              <td>
                <code>{THEME_STORAGE_KEY}</code> (stockage local du navigateur)
              </td>
              <td>Retenir le thème clair si vous l&apos;avez choisi</td>
              <td>Jusqu&apos;à ce que vous reveniez au thème sombre</td>
            </tr>
          </tbody>
        </LegalTable>
      </>
    ),
  },
  {
    id: "securite",
    title: "Sécurité",
    content: (
      <ul>
        <li>Tous les échanges avec le site sont chiffrés (HTTPS).</li>
        <li>Les mots de passe sont hachés avec l&apos;algorithme argon2id.</li>
        <li>
          Le cookie de session est inaccessible aux scripts de la page et n&apos;est envoyé
          qu&apos;en HTTPS.
        </li>
        <li>Les tentatives de connexion répétées sont limitées pour empêcher le piratage.</li>
      </ul>
    ),
  },
  {
    id: "droits",
    title: "Vos droits",
    content: (
      <>
        <p>Vous disposez à tout moment des droits suivants sur vos données :</p>
        <ul>
          <li>
            <strong>accès</strong> : obtenir une copie de vos données ;
          </li>
          <li>
            <strong>rectification</strong> : corriger des données inexactes, notamment depuis{" "}
            <Link href={routes.profile}>votre espace personnel</Link> ;
          </li>
          <li>
            <strong>effacement</strong> : demander la suppression de vos données ;
          </li>
          <li>
            <strong>limitation</strong> et <strong>opposition</strong> : demander qu&apos;on cesse
            d&apos;utiliser vos données, pour des motifs tenant à votre situation ;
          </li>
          <li>
            <strong>portabilité</strong> : récupérer vos données dans un format réutilisable ;
          </li>
          <li>
            <strong>retrait du consentement</strong> : vous désabonner de la newsletter depuis votre
            espace personnel ;
          </li>
          <li>
            <strong>directives post-mortem</strong> : indiquer ce que deviennent vos données après
            votre décès.
          </li>
        </ul>
        <p>
          Pour exercer ces droits, écrivez à{" "}
          <ToComplete>adresse email dédiée (la même qu&apos;en section 1)</ToComplete>. Nous vous
          répondons dans un délai d&apos;un mois. Nous pourrons vous demander de justifier de votre
          identité en cas de doute.
        </p>
      </>
    ),
  },
  {
    id: "reclamation",
    title: "Réclamation auprès de la CNIL",
    content: (
      <p>
        Si vous estimez que vos droits ne sont pas respectés, vous pouvez adresser une réclamation à
        la Commission nationale de l&apos;informatique et des libertés (CNIL), sur{" "}
        <a href="https://www.cnil.fr/fr/plaintes" rel="noopener noreferrer">
          cnil.fr/fr/plaintes
        </a>{" "}
        ou par courrier : CNIL, 3 place de Fontenoy, TSA 80715, 75334 Paris Cedex 07.
      </p>
    ),
  },
  {
    id: "modifications",
    title: "Modifications de cette politique",
    content: (
      <p>
        Nous pouvons mettre à jour cette politique, par exemple si nous ajoutons un service. La date
        de dernière mise à jour figure en haut de la page. En cas de changement important, nous vous
        prévenons par email.
      </p>
    ),
  },
];

export default function PrivacyPage() {
  return (
    <LegalPage
      title="Politique de confidentialité"
      intro={
        <p>
          Cette page explique quelles données personnelles Heros-SOS collecte, pourquoi, combien de
          temps elles sont conservées et comment exercer vos droits.
        </p>
      }
      updatedAt="2026-10-22"
      sections={sections}
    />
  );
}
