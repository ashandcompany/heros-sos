# OpenAPI / Swagger

Deux approches étaient possibles. **Décision : spec d'abord** (`openapi.yaml` et `oapi-codegen`), et le front peut générer son client TypeScript depuis le même fichier.

| Approche                       | Outil                                                                                              | Avantage                                                                                             | Compromis                                                     |
| ------------------------------ | -------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| **Spec d'abord** (recommandée) | Fichier `openapi.yaml` (OpenAPI 3) et `oapi-codegen` pour générer les types et l'interface serveur | Le contrat est la référence, le front peut générer son client, le code ne peut pas dériver de la doc | Il faut écrire le YAML d'abord                                |
| **Code d'abord**               | `swaggo/swag` (annotations en commentaires)                                                        | Rapide à démarrer                                                                                    | Génère surtout du Swagger 2.0, et la doc peut dériver du code |

Dans les deux cas, Swagger UI est servi sur `/docs`, uniquement hors production. Les routes sont préfixées par `/api/v1`, valeur déjà prévue par le README du frontend. Les noms de rôles de l'API (`citizen`, `hero`, `moderator`) devront être alignés avec ceux du frontend (`citoyen`, `super-heros`).
