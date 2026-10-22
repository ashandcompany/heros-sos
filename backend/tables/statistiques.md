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
| Super-vilains mis en prison | `villains` avec `status = 'emprisonne'`, le statut affiché par l'annuaire des vilains (`incident_villains.arrested` n'est écrit par aucune route de l'API) |
| Interventions et vilains emprisonnés d'un héros | `intervention_heroes` (`confirmed`) filtrées sur `hero_id`, jointes à `interventions` puis `incident_villains` (vilains distincts) |
| Note d'un héros | moyenne de `hero_reviews.rating` (`published`) pour son `hero_id` |
| Films d'un héros | `hero_films` avec `status = 'approved'` |
| Pire ennemi | vilain le plus fréquent dans `incident_villains` pour les interventions du héros (via `intervention_heroes`) |

## Compteurs de lecture

Compter sur 5 millions d'incidents coûte de 150 ms à plusieurs secondes selon le filtre : les totaux et les statistiques ci-dessus ne sont plus recalculés à chaque lecture, mais **tenus à jour à chaque écriture** par des triggers de niveau instruction (une seule mise à jour par compteur, même pour 100 000 lignes insérées d'un coup, et aucun chemin d'écriture qui puisse les oublier). `rebuild_read_counters()` les recalcule depuis les tables ; `go run ./cmd/replay -rebuild-counters` l'appelle.

| Table | Une ligne par | Contenu | Sert à |
|---|---|---|---|
| `incident_stats` | `(moderation, status, type_id)` | `incidents`, `citizens_rescued` | Total des listes (sans filtre ou par statut ; le public et les héros lisent la ligne `approved`), pourcentage par type, citoyens secourus |
| `reporter_stats` | `(reporter_id, status)` | `incidents` | Total de « mes incidents » (tous les états de modération : un déclarant voit ses propres incidents en attente) |
| `villain_stats` | `villain_id` | `incidents` (approuvés) | Nombre d'incidents d'un vilain (fiche, liste, total de son historique public) |
| `hero_stats` | `hero_id` | `interventions` (confirmées), `resolved` (réussies, incident approuvé) | Nombre d'interventions d'un héros (fiche, liste) et total de son historique public |

La modération (`incidents.moderation`) est une dimension de `incident_stats`, et `villain_stats` comme `hero_stats.resolved` ne comptent que ce que le public voit (incidents approuvés) : un trigger répercute une décision de modération sur les colonnes `incident_villains.incident_approved` et `intervention_heroes.resolved_at`, donc sur ces compteurs, dans les deux sens.

`reporter_stats` n'a pas de clé étrangère vers `users` : le déclarant d'un incident passe à `NULL` à la suppression de son compte, et la mise à jour qui le fait retire l'incident de son compteur, que la ligne existe encore ou non. Les lignes à zéro restent en place.

Sur cette base, les listes et les fiches se lisent en quelques millisecondes (voir [Performance et synchronisation](../stack/performance-et-synchronisation.md)).

## Tableau de bord des héros

`GET /hero/dashboard` (rôle héros) sert les statistiques ci-dessus, calculées à la demande, plus deux listes. Tout ce
qui vient des incidents (comptes, citoyens secourus, interventions, ennemis, listes) ne retient que ceux dont
`moderation = 'approved'` : un incident en attente ou rejeté n'existe pas pour les héros, même si une intervention
y a déjà été menée.


| Champ de la réponse | Source |
|---|---|
| `global.interventions` | `interventions` terminées (`succeeded` ou `failed`) |
| `global.incidentsByType` | tout le catalogue `incident_types`, y compris les types sans incident, du plus fréquent au moins fréquent |
| `global.rescuedCitizens` | `SUM(incidents.citizens_rescued)` des incidents `succeeded` |
| `global.villainsIdentified`, `villainsImprisoned` | `villains`, dont `status = 'emprisonne'` |
| `personal.successfulInterventions`, `failedInterventions` | `intervention_heroes` (`confirmed`) du héros, jointes à `interventions` |
| `personal.averageRating`, `ratingsCount` | `hero_reviews` publiés du héros |
| `personal.videos`, `publishedVideos` | `hero_films` du héros, dont ceux `approved` |
| `personal.mainEnemies` | 3 vilains les plus fréquents (un compte par incident) parmi les incidents des interventions confirmées du héros |
| `myOngoingIncidents` | incident de l'intervention `in_progress` où le héros est `confirmed` et `is_active` |
| `recentIncidents` | 5 derniers incidents `reported`, `in_progress` ou `failed` (un incident échoué peut être repris) |

Sur plusieurs millions d'incidents, ces agrégats ne se recalculent pas à chaque appel : `global.incidentsByType` et
`global.rescuedCitizens` se lisent dans `incident_stats` (ligne `approved`), et les statistiques (globales, et celles de
chaque héros) passent par un cache en mémoire (`HEROS_DASHBOARD_STATS_TTL`, 30 s) qui sert toujours la dernière valeur
sur-le-champ et se rafraîchit en arrière-plan. Les deux listes d'incidents restent à jour. Le nombre d'interventions
terminées n'a pas de compteur : il se calcule dans ce rafraîchissement. Voir
[Performance et synchronisation](../stack/performance-et-synchronisation.md).
