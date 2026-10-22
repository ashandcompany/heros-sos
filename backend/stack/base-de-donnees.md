# Base de données

PostgreSQL remplace MySQL, accédé avec **GORM** (`gorm.io/gorm` et `gorm.io/driver/postgres`, basé sur `pgx`).

- **Migrations :** fichiers SQL versionnés avec `golang-migrate`. `AutoMigrate` de GORM n'est utilisé qu'en développement, jamais en production (il ne supprime rien et ne se relit pas).
- **Connexion :** TLS, utilisateur applicatif à droits minimaux (pas de superutilisateur).
- **Identifiants :** UUID générés côté Go (v7), pour que l'identifiant de la ligne soit connu avant l'insertion et serve de donnée associée (AAD) au chiffrement des champs.
- **Modèles :** les structs GORM ne sont jamais renvoyées par l'API. Les types générés par OpenAPI (DTO) sont séparés des entités, pour ne pas exposer le hash du mot de passe ou des champs chiffrés.
- **Suppression RGPD :** pas de `gorm.DeletedAt` (suppression logique) sur les données personnelles, car la ligne resterait en base. Suppression réelle (`Unscoped().Delete`) ou anonymisation, pour le droit à l'effacement.
- **Chiffrement des champs :** fait dans le repository ou dans les hooks `BeforeSave` et `AfterFind`, avec `internal/crypto`. Un type `Valuer/Scanner` seul ne connaît pas l'identifiant de la ligne, donc il ne convient pas pour l'AAD.
- **Injection SQL :** GORM paramètre ses requêtes. Le risque vient de `Raw`, `Exec` et `Where` avec une chaîne concaténée : toujours des `?` avec des arguments.
- **Mails (repli outbox, sans RabbitMQ) :** `clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}`.
- **Les sessions, les mails à envoyer et les champs chiffrés** (voir [Sessions](sessions.md), [RabbitMQ](rabbitmq.md) et [Chiffrement des champs](chiffrement.md)) vivent dans cette même base.

Compromis par rapport à `sqlc` : GORM va plus vite à écrire, mais le SQL n'est pas vérifié à la compilation, et `Preload` peut provoquer des requêtes en trop. Rien de bloquant à cette échelle.
