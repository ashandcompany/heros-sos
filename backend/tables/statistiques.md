# Statistiques

Calculées à la demande en SQL.

| Statistique (§5.8) | Source |
|---|---|
| Nombre total d'interventions | `interventions` terminées (une par équipe, pas par héros) |
| Pourcentage des types d'incidents | `incidents` groupés par `type_id` |
| Citoyens secourus | `SUM(incidents.citizens_rescued)` pour `status = 'succeeded'` (une seule fois par incident, même avec plusieurs héros) |
| Citoyens en danger secourus | idem, avec `incident_types.endangers_life = true` |
| Citoyens secourus par un héros | même somme, limitée aux incidents où il est `confirmed` dans une intervention `succeeded` : chaque héros de l'incident est crédité |
| Super-vilains identifiés | `villains` |
| Super-vilains mis en prison | `incident_villains` avec `arrested = true` (vilains distincts) |
| Interventions et vilains emprisonnés d'un héros | `intervention_heroes` (`confirmed`) filtrées sur `hero_id`, jointes à `interventions` puis `incident_villains` (vilains distincts) |
| Note d'un héros | moyenne de `hero_reviews.rating` (`published`) pour son `hero_id` |
| Films d'un héros | `hero_films` avec `status = 'approved'` |
| Pire ennemi | vilain le plus fréquent dans `incident_villains` pour les interventions du héros (via `intervention_heroes`) |

## Compteurs de lecture

Compter sur 5 millions d'incidents coûte de 150 ms à plusieurs secondes selon le filtre : les totaux et les statistiques ci-dessus ne sont plus recalculés à chaque lecture, mais **tenus à jour à chaque écriture** par des triggers de niveau instruction (une seule mise à jour par compteur, même pour 100 000 lignes insérées d'un coup, et aucun chemin d'écriture qui puisse les oublier). `rebuild_read_counters()` les recalcule depuis les tables ; `go run ./cmd/replay -rebuild-counters` l'appelle.

| Table | Une ligne par | Contenu | Sert à |
|---|---|---|---|
| `incident_stats` | `(status, type_id)` | `incidents`, `citizens_rescued` | Total des listes (sans filtre ou par statut), pourcentage par type, citoyens secourus |
| `reporter_stats` | `(reporter_id, status)` | `incidents` | Total de « mes incidents » |
| `villain_stats` | `villain_id` | `incidents` | Nombre d'incidents d'un vilain (fiche, liste, total de son historique) |
| `hero_stats` | `hero_id` | `interventions` (confirmées), `resolved` (réussies) | Nombre d'interventions d'un héros (fiche, liste) et total de son historique |

`reporter_stats` n'a pas de clé étrangère vers `users` : le déclarant d'un incident passe à `NULL` à la suppression de son compte, et la mise à jour qui le fait retire l'incident de son compteur, que la ligne existe encore ou non. Les lignes à zéro restent en place.

Sur cette base, les listes et les fiches se lisent en quelques millisecondes (voir [Performance et synchronisation](../stack/performance-et-synchronisation.md)).
