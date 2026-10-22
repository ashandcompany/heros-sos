# Stack du backend

Complément au cahier des charges, qui prévoit PHP 8.4 et MySQL (§2.4) : le backend est en **Go** avec **PostgreSQL**. Le frontend est en **Next.js** (dossier `frontend/`) et parle à l'API Go en REST. Le schéma des tables est dans [TABLES.md](TABLES.md).

## Les sujets

- [Base de données](stack/base-de-donnees.md) : PostgreSQL et GORM, migrations, identifiants, suppression RGPD, injection SQL.
- [OpenAPI / Swagger](stack/openapi.md) : le contrat d'API (spec d'abord) et Swagger UI hors production.
- [Docker](stack/docker.md) : les services de `docker-compose.yml` (api, worker, proxy, web, db, rabbitmq, mailpit).
- [RabbitMQ](stack/rabbitmq.md) : la file des mails, et le repli sur la table `mail_outbox`.
- [Performance et synchronisation](stack/performance-et-synchronisation.md) : tenir 5 millions d'incidents sous 350 ms (index, compteurs, curseur), le mode hors ligne (file, idempotence), le journal d'événements, le temps réel et le monitoring ELK.
- [Monitoring ELK](stack/elk.md) : le profil Docker Compose `elk` (Elasticsearch, Logstash, Kibana, Filebeat) : démarrer, ce qu'on y voit, vérifier, dépanner.
- [OMDb pour les films](stack/omdb.md) : le catalogue de films, son quota, sa licence, la clé et le cache.
- [Sécurité et chiffrement](stack/securite.md) : les mesures retenues (cascade AES-256 puis XChaCha20, TLS hybride, Shamir, Ed25519).
- [Chiffrement des champs](stack/chiffrement.md) : le détail de la cascade à clé de 512 bits (clés, format stocké, limites).
- [Sessions](stack/sessions.md) : jeton, cookie, endpoints d'authentification, durées et règles (2FA, CSRF).
- [AES, RSA et protocoles](stack/aes-rsa.md) : annexe sur les variantes, les modes d'utilisation et les protocoles qui emploient AES et RSA.

## Décisions prises

- **OpenAPI :** spec d'abord (`oapi-codegen`).
- **Cookie de session :** option A, Next relaie le `Set-Cookie` de Go.
- **Devenir héros :** formulaire dédié « Devenir super-héros » dans « Mon compte », validé par un modérateur.
- **2FA :** TOTP, obligatoire pour les modérateurs avant toute mise en ligne (voir [Sessions](stack/sessions.md)).
- **Newsletter :** double opt-in pour les visiteurs anonymes.
- **Appareil de la session :** résumé (`device_label`), pas de User-Agent brut.
- **Chiffrement :** nom, prénom, date de naissance, téléphone et adresse de tous les utilisateurs.
- **« Citoyen vérifié » :** information de confiance, sans droit supplémentaire.
- **RabbitMQ** pour les mails, si l'hébergeur supporte Docker.

## Décisions en attente

1. **Proximité des héros :** à décider avec le client. Si elle est retenue : la position actuelle du héros (là où il se trouve, pas son domicile), partagée volontairement et visible des modérateurs, en clair, sans historique. Voir `hero_locations` dans [tables/comptes.md](tables/comptes.md#hero_locations-optionnelle).
2. **Encyclopédie Marvel** (§6.1.4) : source à vérifier, l'API Marvel semble fermée.

## Prochaines étapes

- Mettre à jour la section Stack du `README.md` (Go, PostgreSQL, Docker, OpenAPI, OMDb, plus RabbitMQ).
- Initialiser `backend/` : `go mod init`, structure, `Dockerfile`, `docker-compose.yml` et `openapi.yaml`, avec `/health` comme premier endpoint.
