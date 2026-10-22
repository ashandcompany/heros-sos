# Docker

Un `docker-compose.yml` avec ces services :

- `api` : image Go en multi-stage, avec une image finale minimale.
- `worker` : worker Go qui lit la file `mail.send` et envoie les mails.
- `proxy` : Caddy ou Traefik. Il termine le TLS et route `/api/*` vers `api` et le reste vers Next.js, pour que le navigateur ne parle qu'à une seule origine (voir [Sessions](sessions.md)).
- `web` : Next.js, si le frontend est aussi conteneurisé (à confirmer avec Emma).
- `db` : PostgreSQL, avec un volume.
- `rabbitmq` : retenu pour les mails, avec l'interface de management (à condition que l'hébergeur supporte Docker).
- `mailpit` : faux serveur SMTP pour voir les mails en développement.
- `elasticsearch`, `logstash`, `kibana`, `filebeat` et `elk-setup` dans le profil `elk` (`make elk`) : le monitoring ELK de développement, qui lit les journaux JSON de l'API (voir [Performance et synchronisation](performance-et-synchronisation.md)). Ils pèsent 3 à 4 Go de mémoire : rien ne démarre avec `make up`.
- `adminer` ou pgAdmin dans un profil `dev` uniquement : le cahier §2.4 autorise phpMyAdmin seulement en développement et interdit tout outil équivalent en production (RGPD).

Les secrets passent par `.env`, avec un `.env.example` commité et jamais le vrai fichier.
