---
title: Architecture
---

Heros-SOS tourne dans quatre conteneurs Docker : `proxy`, `web`, `api` et `db`. En développement s'y ajoutent
Mailpit (les mails), RabbitMQ et ce site de documentation. Cette page montre qui parle à qui ; la page suivante, [Une connexion de bout en bout](architecture/connexion.md), suit une connexion.

## Vue d'ensemble

```mermaid
flowchart TB
    nav(["<b>Navigateur</b>"])

    subgraph edge["Réseau edge"]
        proxy["<b>proxy</b><br/>Caddy<br/>dev : 8088<br/>prod : 80 et 443 (TLS)"]
        web["<b>web</b><br/>Next.js<br/>port interne 3000"]
        api["<b>api</b><br/>Go<br/>port interne 8080"]
    end

    subgraph backend["Réseau backend (fermé en production)"]
        db[("<b>db</b><br/>PostgreSQL 18")]
        mailpit["<b>mailpit</b><br/>faux SMTP (dev)"]
        rabbit["<b>rabbitmq</b><br/>pas encore utilisé"]
    end

    subgraph externes["Services externes"]
        smtp["Serveur SMTP<br/>(production)"]
        omdb["OMDb<br/>catalogue de films"]
        geo["Géoplateforme<br/>adresses"]
        osm["OpenStreetMap<br/>tuiles de la carte"]
    end

    nav -->|"HTTPS"| proxy
    nav -->|"tuiles"| osm
    proxy -->|"/api/v1/*"| api
    proxy -->|"toutes les autres pages"| web
    web -->|"API_URL + cookie de session"| api
    web -->|"autocomplétion /api/adresses"| geo
    api --> db
    api -->|"dev"| mailpit
    api -->|"prod"| smtp
    api -->|"fiches de films"| omdb
```

À retenir :

- **Un seul point d'entrée.** Le navigateur ne voit que le proxy Caddy : **http://localhost:8088** en
  développement, 80 et 443 en production. Il sert le front Next.js et route `/api/v1/*` vers l'API : une seule
  origine, donc un cookie de session sans CORS.
- **Les ports 3000 (front) et 8080 (API) sont internes** au réseau Docker : ils ne sont pas publiés, donc
  inutilisables depuis la machine de développement. Le message `Local: http://localhost:3000` des journaux du conteneur `web`
  est celui de Next.js lui-même : ce lien ne mène nulle part, il faut passer par le proxy. (En développement,
  l'API est aussi joignable directement sur `localhost:8081`, pour `curl` et les outils.)
- **Le front appelle l'API depuis son serveur**, pas depuis le navigateur (Server Actions, rendu serveur),
  directement sur le réseau `edge`, sans repasser par le proxy.
- **La base n'est jamais exposée.** Elle est sur le réseau `backend`, fermé en production : ni la base ni l'API
  n'ont de port publié (seul le proxy en a), et la base n'a aucun accès à Internet.
- **Les mails passent par l'API.** En développement ils partent vers Mailpit (http://localhost:8025) ; en
  production vers un vrai serveur SMTP chiffré. La file d'envoi est dans le processus de l'API : RabbitMQ est
  démarré mais aucun code ne l'utilise encore.
- **Deux volumes à sauvegarder** : `pgdata` (la base) et `avatars` (les photos de profil).
