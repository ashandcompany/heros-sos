# Monitoring ELK

**Un profil Docker Compose, `elk`, ajoute à la pile de développement de quoi voir les journaux de l'API dans Kibana : les temps de réponse par route, les erreurs, et chaque action sur un incident ou une intervention.** Il ne démarre jamais tout seul : `make up` ne lance rien de plus, il faut `make elk`.

> **Statut.** La pile a été **démarrée et vérifiée le 22/10/2026** sur Docker Desktop (macOS, 7,5 Go alloués), version 9.3.1. Constaté : les cinq services démarrent (environ 3 minutes la première fois, téléchargement des images compris), `elk-setup` importe le dashboard, les lignes de requêtes et les événements métier arrivent dans leurs index, le p95 par route se calcule, et les sept graphiques du dashboard s'affichent avec des données. Cette vérification a révélé un défaut, corrigé : l'adresse IP restait dans `event.original` (voir [Ce que le monitoring ne doit pas faire](#ce-que-le-monitoring-ne-doit-pas-faire)). **Non vérifié** : Linux (`vm.max_map_count`), Docker Desktop sous Windows, une autre version d'Elastic que 9.3.1. La section [Dépannage](#dépannage) liste ce qui peut coincer. Le contexte (pourquoi ELK, ce qui est mesuré) est dans [Performance et synchronisation](performance-et-synchronisation.md).

## Le trajet d'une ligne de journal

```mermaid
flowchart LR
    API["API<br/>journaux JSON, service=api"] --> Docker["Docker<br/>(driver json-file)"]
    Docker --> Filebeat
    Filebeat --> Logstash["Logstash<br/>trie, retire le personnel"]
    Logstash --> ES[("Elasticsearch<br/>deux index, 7 jours")]
    ES --> Kibana["Kibana<br/>dashboard et Discover"]
```

1. L'API écrit **une ligne JSON par requête** (`route`, `status`, `duration_ms`, `request_id`) et **une par événement métier** (`event_type`, `incident_id`, `command_id`, `sync_delay_ms`…). Le format JSON est réglé par `make elk` (`HEROS_LOG_FORMAT=json`) ; sans lui, les journaux restent en texte lisible et rien n'arrive dans Kibana.
2. **Filebeat** lit les fichiers de journaux des conteneurs (`/var/lib/docker/containers`, en lecture seule) et ne garde que ceux de l'API.
3. **Logstash** reconnaît le type de ligne, supprime les champs personnels (dont l'adresse IP) et aiguille vers l'un des deux index.
4. **Elasticsearch** les stocke, **Kibana** les affiche.

## Prérequis

| Quoi | Détail |
|---|---|
| Mémoire | Mesuré : l'ELK prend **3,3 Go** (Elasticsearch 1,5 ; Kibana 0,9 ; Logstash 0,8 ; Filebeat 0,1), soit **5,7 Go avec toute la pile** : donner **8 Go** à Docker Desktop (Settings, Resources) |
| Disque | Les images pèsent **environ 6 Go** sur le disque de Docker (Kibana 2,3 ; Elasticsearch 1,6 ; Logstash 1,4 ; Filebeat 0,6), les données un peu plus. Le Makefile mesure l'espace libre dans le **disque virtuel de Docker** (pas celui de la machine) et refuse de démarrer sous **8 Go** tant qu'une image manque (`ELK_DISK_MIN_MB`), sous **2 Go** ensuite (`ELK_DISK_MIN_MB_READY`). Docker Desktop affiche cet espace en bas de sa fenêtre (« Disk : … used (limit …) ») |
| Fichier `.env` | `backend/.env` doit exister (`make env`) : il existe déjà si la pile de développement tourne |
| Linux seulement | Elasticsearch exige `sudo sysctl -w vm.max_map_count=262144` |
| Réseau | Les images viennent de `docker.elastic.co` (gratuites en auto-hébergement, offre Basic) |

## Démarrer et arrêter

```bash
make elk            # démarre le monitoring ; Kibana : http://localhost:5601
make elk-down       # l'arrête et remet les journaux de l'API en texte
make elk-down V=1   # idem, et supprime aussi les données d'Elasticsearch (volume esdata)
```

`make elk` fait trois choses :

- il vérifie l'espace disque libre ;
- il **recrée le conteneur `api`** avec des journaux JSON (il reprend le texte lisible au prochain `make up` ou `make elk-down`) ;
- il démarre `elasticsearch`, `logstash`, `kibana`, `filebeat` et la tâche `elk-setup`.

Le premier démarrage prend **une à deux minutes** : Elasticsearch puis Kibana doivent être prêts avant que `elk-setup` ne crée les index et le dashboard. `elk-setup` s'arrête seul quand il a fini, ce qui est normal. Il est idempotent : relancer `make elk` ne casse rien.

Le profil se manipule aussi à la main : c'est `--profile elk` qui fait apparaître ces services, sans lui Compose les ignore.

```bash
docker compose -p heros-sos-dev --env-file backend/.env -f docker-compose.yml -f docker-compose.dev.yml \
  --profile elk ps
```

## Les services

| Service | Image | Rôle | Port sur la machine |
|---|---|---|---|
| `elasticsearch` | `docker.elastic.co/elasticsearch/elasticsearch` | Stockage et recherche, un seul nœud, 1 Go de mémoire Java | `127.0.0.1:9200` (`ELASTICSEARCH_PORT`) |
| `logstash` | `docker.elastic.co/logstash/logstash` | Tri des lignes, suppression du personnel, aiguillage vers deux index, 512 Mo de mémoire Java | aucun (réseau interne, port 5044) |
| `kibana` | `docker.elastic.co/kibana/kibana` | Dashboard et exploration | `127.0.0.1:5601` (`KIBANA_PORT`) |
| `filebeat` | `docker.elastic.co/beats/filebeat` | Lit les journaux des conteneurs (tourne en root pour y accéder) | aucun |
| `elk-setup` | image de Kibana | Tâche unique : cycle de vie de 7 jours, modèles d'index, vues de données, dashboard | aucun |

Les quatre images sont **à la même version**, fixée par `ELASTIC_VERSION` (9.3.1 par défaut, jamais `latest`). Pour la changer : `ELASTIC_VERSION=… make elk`. Tous les ports sont liés à `127.0.0.1` : rien n'est joignable depuis le réseau.

Les fichiers de configuration sont dans `backend/docker/elk/` (Filebeat, pipeline Logstash, modèles d'index, politique de cycle de vie, dashboard, `setup.sh`).

## Ce qu'on y trouve

Deux index, qui ne gardent que **7 jours** de données :

| Index | Contenu | Vue de données Kibana |
|---|---|---|
| `herossos-logs-*` | Une ligne par requête (`route`, `status`, `duration_ms`, `request_id`) et les journaux de l'application | `herossos-logs` |
| `herossos-events-*` | Une ligne par action sur un incident ou une intervention (`event_type`, `aggregate_type`, `incident_id`, `command_id`, `sync_delay_ms`) | `herossos-events` |

Un événement a pour identifiant de document celui de l'événement : la journalisation est « au moins une fois », et une ligne envoyée deux fois remplace la première au lieu de s'ajouter.

Le dashboard **« Heros-SOS : performance et synchronisation »** (Kibana, Dashboards) montre :

1. le p95 de `duration_ms` par route (les cinq plus lentes), le p95 global et le nombre de requêtes ;
2. les erreurs et les conflits (statut 400 et plus) ;
3. les événements métier par minute ;
4. le délai de synchronisation des actions faites hors ligne (`sync_delay_ms`) ;
5. la chronologie des événements (recherche enregistrée).

Le flux temps réel (`/events/stream`) reste ouvert jusqu'à 10 minutes : il est écarté des durées, sinon il fausserait tous les p95.

## Vérifier que ça marche

```bash
# 1. Les conteneurs tournent (elk-setup, lui, s'est arrêté) et Elasticsearch répond
curl 'http://localhost:9200/_cluster/health?pretty'          # "status" : "green" ou "yellow"

# 2. Générer du trafic à travers le proxy de développement
for i in $(seq 1 20); do curl -s -o /dev/null http://localhost:8088/api/v1/heroes; done

# 3. Les lignes sont arrivées (quelques secondes de délai)
curl 'http://localhost:9200/herossos-logs-*/_count?pretty'   # "count" supérieur à 0
```

Puis dans Kibana : **Discover**, vue de données `herossos-logs`, et le dashboard. Pour voir un événement métier, déclarer ou modifier un incident dans l'application, puis ouvrir la vue `herossos-events`.

## Dépannage

Ces causes sont celles qu'on attend ; elles n'ont pas encore été confirmées sur une vraie machine.

| Symptôme | Cause probable | Que faire |
|---|---|---|
| `make elk` s'arrête sur « Le disque de Docker est presque plein » | Pas assez de place dans le disque virtuel de Docker (8 Go au premier démarrage, pour les images ; 2 Go ensuite) | `docker builder prune -a -f` (cache de build, sans danger) ; ou agrandir le disque durablement (Docker Desktop, Settings, Resources, Virtual disk limit) ; ou continuer malgré tout avec `FORCE=1 make elk` |
| Elasticsearch redémarre en boucle ou est tué | Mémoire de Docker Desktop trop basse (OOM) ; sous Linux, `vm.max_map_count` trop bas | Passer Docker à 6 à 8 Go ; régler `vm.max_map_count` |
| `elk-setup` signale « Délai dépassé » | Elasticsearch ou Kibana n'ont pas fini de démarrer (il attend jusqu'à 5 minutes) | Relancer `make elk` (idempotent) |
| Kibana s'ouvre mais le dashboard manque | L'import du dashboard, écrit à la main, a échoué (non bloquant) | Voir `docker logs heros-sos-dev-elk-setup-1`. Les vues de données restent utilisables dans Discover, et les graphiques se recréent dans Lens |
| Kibana ne montre aucune donnée | L'API écrit encore en texte, ou aucun trafic n'a eu lieu | Relancer `make elk` (recrée l'API en JSON), puis générer du trafic. Regarder `docker logs heros-sos-dev-filebeat-1` |
| Les journaux de l'API sont illisibles dans le terminal | Ils sont en JSON pendant que l'ELK tourne | `make elk-down` remet le texte |

## Ce que le monitoring ne doit pas faire

- **Données personnelles.** Les adresses et descriptions d'incidents n'entrent jamais dans le journal d'événements ni dans ELK : les événements ne portent que des identifiants et des états, et Logstash retire en plus l'adresse IP et tout champ libre qui s'y glisserait. Attention à `event.original` : Filebeat y garde la **ligne brute** (adresse IP et payload compris), que retirer les champs un par un ne couvre pas. Logstash le supprime explicitement. Vérifié au premier démarrage : sans cela, l'adresse IP de chaque requête restait stockée.
- **Sécurité : démonstration locale uniquement.** Elasticsearch démarre avec la sécurité **désactivée** (pas de mot de passe, pas de TLS), Filebeat tourne en root, et le seuil de remplissage du disque d'Elasticsearch est désactivé pour qu'un portable presque plein ne mette pas les index en lecture seule. Rien de cela n'a sa place en production : il y faudrait la sécurité activée, des mots de passe dans des fichiers de secrets et TLS.
- **Pas dans la CI.** Cette pile n'est jamais lancée par les tests.
