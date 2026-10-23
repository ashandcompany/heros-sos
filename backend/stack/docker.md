# Docker

Un `docker-compose.yml` avec ces services :

- `api` : image Go en multi-stage, avec une image finale minimale.
- `mail-worker` : même image que `api`, binaire `mailworker` (`cmd/mailworker`). Il lit la file `mail.send` et envoie les mails. Il n'a ni la base ni les clés de chiffrement. Voir [RabbitMQ](rabbitmq.md).
- `proxy` : Caddy ou Traefik. Il termine le TLS et route `/api/*` vers `api` et le reste vers Next.js, pour que le navigateur ne parle qu'à une seule origine (voir [Sessions](sessions.md)).
- `web` : Next.js, si le frontend est aussi conteneurisé (à confirmer avec Emma).
- `db` : PostgreSQL, avec un volume. En développement, `docker-compose.dev.yml` le démarre avec `shared_preload_libraries=pg_stat_statements` : l'extension classe les requêtes par temps total, et `make slow-queries` les affiche (voir [Performance et synchronisation](performance-et-synchronisation.md)). Rien de tel en production.
- `rabbitmq` : la file des mails, avec l'interface de management (à condition que l'hébergeur supporte Docker). En production, profil `rabbitmq` (avec `mail-worker`).
- `mailpit` : faux serveur SMTP pour voir les mails en développement.
- `elasticsearch`, `logstash`, `kibana`, `filebeat` et `elk-setup` dans le profil `elk` (`make elk`) : le monitoring ELK de développement, qui lit les journaux JSON de l'API (voir [Monitoring ELK](elk.md) et [Performance et synchronisation](performance-et-synchronisation.md)). Mesuré, ils prennent 3,3 Go de mémoire et leurs images pèsent environ 6 Go : rien ne démarre avec `make up`.
- `adminer` ou pgAdmin dans un profil `dev` uniquement : le cahier §2.4 autorise phpMyAdmin seulement en développement et interdit tout outil équivalent en production (RGPD).

Les secrets passent par `.env`, avec un `.env.example` commité et jamais le vrai fichier.
