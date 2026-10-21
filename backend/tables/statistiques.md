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
