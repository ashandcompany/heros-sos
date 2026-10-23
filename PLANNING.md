# Planning et suivi du projet

**Ce document est reconstitué le 23/10/2026** à partir de l'historique Git et des pull requests (dates de fusion, auteurs, titres). Ce n'est pas un planning écrit avant le travail : il dit ce qui a été livré et quand, comment le travail était réparti, ce qui a changé en route, et ce qui reste. Le dépôt n'a ni issues GitHub ni milestones : le suivi vivait dans les descriptions de PR et dans les documents de contrat ([backend/CONTRAT-MAXIME.md](backend/CONTRAT-MAXIME.md), [frontend/docs/backend-contract.md](frontend/docs/backend-contract.md)).

Légende : ✅ fait · 🟡 fait mais à finir ou à décider · ⬜ pas fait.

## 1. Équipe et périmètres

Les périmètres sont ceux de [`.github/CODEOWNERS`](.github/CODEOWNERS).

| Qui | Compte GitHub | Périmètre |
|---|---|---|
| **Maxime (Dev A)** | `@le-veilleur` | Identité et sécurité du backend : comptes, sessions, 2FA, chiffrement, mails, newsletter, audit, modération des comptes. Puis performance, synchronisation hors ligne, journal d'événements, ELK, tableau de bord des héros, RabbitMQ, outillage Docker/Make, documentation technique |
| **Dev B** | `@Jersey276` | Incidents, interventions, vilains, avis, films, blog, proximité des héros, socle Docker et CI |
| **Emma** | `@ashandcompany` | Frontend Next.js et son branchement sur l'API |

Pull requests ouvertes sur le dépôt : 56 (Emma 20, Maxime 19, Dev B 17) ; 53 fusionnées, 3 fermées sans fusion.

## 2. Chronologie

Cinq jours de travail, du lundi 19 au vendredi 23 octobre 2026. 293 commits (hors fusions) sur `develop`. Les numéros renvoient aux pull requests.

| Jour | Jalon | Livré | Commits (Maxime / Emma / Dev B) |
|---|---|---|---|
| **Lun 19/10** | Socle | Dépôt, CI (#5), Docker (#7), comptes, sessions, sécurité et modération côté backend (#8), incidents, vilains et interventions (#10). Frontend : accueil, espace citoyen, espace héros, administration des super-héros (#1 à #4, #9) | 47 / 27 / 5 |
| **Mar 20/10** | Environnements et contrat | Environnements Docker de développement et de production, PostgreSQL 18, Swagger UI (#12, #13), Compose unifié (#14, #15), API adaptée aux besoins du front (#16), fiches des personnages (#11), pages légales (#17, #18) | 4 / 21 / 6 |
| **Mer 21/10** | Compte et branchement | Profil, contact, newsletter et site de documentation (#25), 2FA des modérateurs, confirmation d'email, désinscription (#32), blog (#26), avis et vidéothèque (#20, #30), fiches des héros (#23), incidents publics (#29), CODEOWNERS, CI de la documentation, déploiement de la documentation sans Pages (#33). Première livraison `develop → main` (#19). Branchement du front sur l'API, domaine par domaine (branche `feat/api-tooling`, #22, #24, #27, #31) | 49 / 25 / 10 |
| **Jeu 22/10** | Défi technique | Modération des incidents (#39), statistiques d'administration (#38), tableau de bord des héros (#40), **performance à 5 M d'incidents, synchronisation hors ligne, journal d'événements et ELK (#41)**, proximité des héros (#43), correction ELK (#45, #46), mesures à 5 M (#48), préchauffage du cache (#49), `golangci-lint` en CI (#36) | 32 / 22 / 9 |
| **Ven 23/10** | Fiabilité et livraison | File RabbitMQ et `mail-worker` (#54), `./make` pour les machines sans make (#51), 2FA pour tous les comptes (#56), messages de newsletter (#50), réécriture du README | 27 / 7 / 2 |

## 3. Écarts et imprévus

### 3.1 Le périmètre a bougé

[backend/MVP.md](backend/MVP.md) rangeait plusieurs sujets en « Hors MVP (version 2) ». Ils ont été livrés avant la fin.

| Prévu hors MVP | État | PR |
|---|---|---|
| Notes et réponses des héros, films et OMDb | ✅ | #20, #30 |
| Articles du blog | ✅ | #26 |
| Statistiques et tableau de bord | ✅ | #38, #40 |
| Équipes de héros (inviter, rejoindre, proposer et voter un résultat) | ✅ (routes `/interventions/{id}/invitations` et suivantes du contrat) | |
| Envoi des numéros de la newsletter | ✅ (sans suivi par destinataire) | #50 |
| Proximité des héros (`hero_locations`) | ✅ | #43 |
| 2FA des héros et des citoyens | ✅ | #56 |
| RabbitMQ pour les mails (le MVP prévoyait une goroutine) | ✅ | #54 |
| Encyclopédie Marvel | ⬜ la source de données est à vérifier ([points ouverts](backend/tables/points-ouverts.md)) | |

### 3.2 Imprévus rencontrés

Chacun est documenté là où il a été corrigé.

| Quand | Imprévu | Correction | Où lire |
|---|---|---|---|
| Jeu 22/10 | `golang-migrate` n'applique que les migrations au-dessus de la version courante : une migration numérotée plus bas est ignorée en silence. Symptôme relevé dans ELK le 22/10 vers 14 h 38 (UTC) : cinq erreurs 500 sur `/incidents`, dont `column "title" of relation "incidents" does not exist`, le signal décrit dans la documentation | Les migrations de la performance sont renumérotées après toutes celles de `develop` ; règle et procédure de diagnostic écrites | [base-de-donnees.md](backend/stack/base-de-donnees.md), [performance-et-synchronisation.md](backend/stack/performance-et-synchronisation.md) |
| Jeu 22/10 | Un trigger de niveau instruction se déclenche même quand l'`UPDATE` ne touche aucune ligne : celui qui écrit dans sa propre table se rappelait à l'infini (`stack depth limit exceeded`) | Garde en tête de fonction | performance-et-synchronisation.md, P1 |
| Jeu 22/10 | Premier démarrage réel d'ELK : l'adresse IP du client restait dans `event.original` | Champ retiré par Logstash (#45) | [elk.md](backend/stack/elk.md) |
| Jeu 22/10 | À 5 M d'incidents, le premier appel du tableau de bord d'un héros calculait en direct (environ 11 s) | Cache avec préchauffage au démarrage (#49) | performance-et-synchronisation.md |
| Jeu 22/10 | Des pics de requêtes épuisaient les ports locaux (`can't assign requested address`) | Connexions inactives du pool portées de 5 à 25 | performance-et-synchronisation.md, P1 |
| Jeu 22/10 | Les routes ajoutées par `develop` (modération, tableau de bord, tri par résolution) dégradaient les mesures déjà faites | Compteurs et index rendus sensibles à la modération, `incidents.resolved_at`, mesures refaites | performance-et-synchronisation.md, « Mise à jour après la modération » |

## 4. Suivi et organisation

- **Branches.** Travail sur des branches `feat/…`, `fix/…`, `docs/…`, `ci/…`, fusionnées dans `develop` par pull request. Messages de commit au format `type(portée) : résumé`.
- **Contrat avant le code.** Le contrat d'API (`backend/openapi/openapi.yaml`) est la référence commune : les tests du backend échouent si une route diverge, et le frontend vérifie ses types générés (`npm run api:types:check`).
- **Qui fait quoi.** `CODEOWNERS` fixe les périmètres, et `CONTRAT-MAXIME.md` répartit les opérations du contrat entre les deux développeurs.
- **Intégration continue.** Trois workflows (backend, frontend, documentation), verts sur les dernières PR.
- **Ce qui manque.** Pas d'issues, pas de milestones, pas de tableau de bord de tâches : le reste à faire ci-dessous remplace ce suivi.

## 5. Reste à faire (23/10/2026)

Les « Source » indiquent où le point est établi. Les propriétaires sont proposés d'après `CODEOWNERS` et sont **à confirmer**.

### 5.1 Livraison

| # | Point | État | Propriétaire proposé | Source |
|---|---|---|---|---|
| 1 | Fusionner `develop` dans `main` et poser un tag : `main` a 243 commits de retard et ne contient ni la performance, ni ELK, ni le hors ligne. Les PR #34 et #55 ont été fermées sans fusion | ⬜ décision à prendre | Équipe | historique Git |
| 2 | Publier le site de documentation (activer GitHub Pages) | ⬜ | Maxime | [docs-site/README.md](docs-site/README.md) |
| 3 | Environnement de recette et déploiement (rien n'est déployé) | ⬜ | Dev B et Maxime | [CONTRAT-MAXIME.md](backend/CONTRAT-MAXIME.md) §3.3, [README](README.md) |
| 4 | Maquettes Figma : lien à ajouter au README | ⬜ | Emma | [README](README.md) |
| 5 | Activer RabbitMQ en production (`COMPOSE_PROFILES=rabbitmq`, `HEROS_MAIL_QUEUE=rabbitmq`), puis surveiller la file morte | 🟡 | Maxime | CONTRAT-MAXIME.md §3.2 point 8 |

### 5.2 Défi technique

| # | Point | État | Propriétaire proposé | Source |
|---|---|---|---|---|
| 6 | Recréer la base de 5 M d'incidents (`make perf-prepare`, `make seed-perf`, environ 7 minutes et 11 Go libres) et refaire `make perf` avant la démonstration | ⬜ | Maxime | [performance-et-synchronisation.md](backend/stack/performance-et-synchronisation.md) |
| 7 | Jouer le parcours hors ligne dans un vrai navigateur (service worker, IndexedDB, `EventSource`) et l'ajouter aux tests Playwright | ⬜ | Maxime et Emma | performance-et-synchronisation.md §4 |
| 8 | Envoyer `expectedVersion` depuis l'interface (l'API le gère déjà) | ⬜ | Emma | performance-et-synchronisation.md §6 point 4 |
| 9 | Enregistrer une vidéo de secours de la démonstration | ⬜ | Maxime | performance-et-synchronisation.md §3 |
| 10 | Vérifier ELK sous Linux et avec plusieurs instances de l'API ; sécuriser Elasticsearch avant toute mise en ligne | ⬜ | Maxime | performance-et-synchronisation.md §4 et §5 |

### 5.3 Produit et sécurité

| # | Point | État | Propriétaire proposé | Source |
|---|---|---|---|---|
| 11 | Droit à l'effacement et export des données (RGPD) : aucune route ni commande | ⬜ | Maxime | CONTRAT-MAXIME.md §3.2 point 1 |
| 12 | Changer son mot de passe quand on est connecté | ⬜ | Maxime et Emma | CONTRAT-MAXIME.md §3.2 point 2 |
| 13 | Modifier son pseudo et sa description de héros | ⬜ à confirmer avec le cahier | Maxime | CONTRAT-MAXIME.md §3.2 point 3 |
| 14 | Exiger une 2FA récente pour les actions sensibles ; route de lecture du journal d'audit ; suppression de la photo de profil | ⬜ | Maxime | CONTRAT-MAXIME.md §3.2 points 4 à 6 |
| 15 | Limiteur de débit partagé avant de lancer plusieurs instances de l'API | 🟡 | Maxime | CONTRAT-MAXIME.md §3.2 point 7 |
| 16 | Écran des numéros de la newsletter (le contrat existe) | ⬜ | Emma | [backend-contract.md](frontend/docs/backend-contract.md) |
| 17 | Encyclopédie Marvel : trouver une source de données | ⬜ | Équipe | [points-ouverts.md](backend/tables/points-ouverts.md) |

### 5.4 Qualité et documentation

| # | Point | État | Propriétaire proposé | Source |
|---|---|---|---|---|
| 18 | Dette `golangci-lint` du code de Dev B et fichiers mal formatés (à recompter : la liste date du 21/10) | 🟡 | Dev B | CONTRAT-MAXIME.md §3.4 |
| 19 | `internal/herolocation/handler.go` commence par un BOM UTF-8 : `go test -cover` échoue sur le module entier (« invalid BOM in the middle of the file ») | ⬜ | Dev B | constaté le 23/10/2026 |
| 20 | Rafraîchir les documents périmés : `CONTRAT-MAXIME.md` (état du 21/10, 91 opérations au lieu de 110), `MVP.md` (section « À vérifier avec Emma »), `STACK.md` (« Prochaines étapes »), compte des tests, section « Ce qui n'a pas été vérifié » de la page performance (ELK tourne désormais en continu) | ⬜ | Maxime | relecture du 23/10/2026 |
| 21 | Couverture de tests faible sur `film` et `intervention` | 🟡 | Dev B | `go test -cover` du 23/10/2026 |
