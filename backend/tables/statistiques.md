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

Les agrégats globaux parcourent `incidents` : sur plusieurs millions de lignes, il faudra les remplacer par des
compteurs maintenus à l'écriture (trigger ou table de statistiques) plutôt que de les recalculer à chaque appel.
