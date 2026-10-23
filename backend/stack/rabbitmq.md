# RabbitMQ : utile ou pas ?

**Oui, mais seulement pour les mails.** Le cahier §4.3 déclenche des envois groupés : un incident prévient tous les modérateurs et tous les héros, et la newsletter part vers tous les utilisateurs. Sans file, la requête HTTP attend le SMTP. Elle échoue ou expire si le serveur de mail est lent, et un mail raté n'est pas rejoué.

Proposition :

- une file `mail.send` avec un worker Go séparé, qui retente en cas d'échec et envoie les échecs définitifs dans une dead-letter queue ;
- aucune autre file pour l'instant. Les statistiques se calculent à la demande en SQL.

Deux points à connaître :

- **Hébergement :** le cahier §2.1 dit que chaque groupe héberge chez son hébergeur. Un hébergement mutualisé ne fera pas tourner RabbitMQ. Vérifier que l'hébergeur supporte Docker.
- **Alternative sans RabbitMQ :** une table `mail_outbox` en PostgreSQL lue par un worker (`SELECT … FOR UPDATE SKIP LOCKED` évite que deux workers prennent le même mail). Il n'y a rien de plus à héberger et c'est suffisant à cette échelle. RabbitMQ est un peu surdimensionné ici, mais il se justifie pour la démonstration technique ou si d'autres événements sont prévus.

Avec RabbitMQ, le message est publié après le commit en base, avec confirmation de publication. Sinon un crash entre les deux ferait perdre un mail.

**Décision :** RabbitMQ est retenu pour les mails, tant que l'hébergeur supporte Docker. Sinon, repli sur la table `mail_outbox` (voir [tables/technique.md](../tables/technique.md)).

## Mise en œuvre

Le code est dans `internal/mail` (`publisher.go`, `worker.go`, `rabbitmq.go`) et dans `cmd/mailworker`. Les appelants ne changent pas : ils ne connaissent que l'interface `mail.Mailer`, que `mail.Publisher` implémente comme `mail.Queue`.

```
API ── Publisher ──► échange « mail » ──► file « mail.send » ──► mail-worker ──► SMTP
                                               │
                                  échec définitif (après MaxAttempts) ou message illisible
                                               ▼
                                   échange « mail.dead » ──► file « mail.dead »
```

Les mails passent par RabbitMQ quand `HEROS_MAIL_QUEUE=rabbitmq`. Le défaut est `memory` (l'ancienne file dans l'API). Toute la pile de développement utilise RabbitMQ ; en production, le profil Compose `rabbitmq` démarre le courtier et le worker (voir `.env.production.example`).

| Réglage | Rôle |
|---|---|
| `HEROS_MAIL_QUEUE` | `memory` ou `rabbitmq`. |
| `HEROS_RABBITMQ_URL` | Adresse du courtier, avec son mot de passe : en production `HEROS_RABBITMQ_URL_FILE` (secret `rabbitmq_url`, créé par `make secrets`). Jamais journalisée. |
| `HEROS_RABBITMQ_CONFIRM_TIMEOUT` | Attente de la confirmation du courtier après un dépôt (5 s). Au-delà, le mail est refusé. |
| `HEROS_RABBITMQ_RECONNECT_DELAY` | Délai entre deux tentatives de connexion (2 s). |
| `HEROS_MAIL_WORKERS`, `_MAX_ATTEMPTS`, `_RETRY_DELAY`, `_SEND_TIMEOUT` | Pour le worker : envois simultanés (c'est aussi son *prefetch*), nombre d'essais, délai croissant entre deux essais, durée d'un envoi. |

### Ce qui est garanti

- **Un mail accepté est durable.** `Publisher.Send` ne retourne `nil` qu'une fois que RabbitMQ a confirmé (*publisher confirm*) un message persistant dans une file durable. Si l'API ou le courtier s'arrête ensuite, le mail est toujours là.
- **Un mail n'est retiré de la file qu'une fois envoyé** (acquittement après le SMTP). Si le worker est tué en pleine remise, le courtier la redonne : au pire un mail arrive **en double**, jamais zéro. Un envoi en double peut aussi venir d'un dépôt dont la confirmation a expiré puis a été réessayé.
- **Les échecs sont réessayés** `HEROS_MAIL_MAX_ATTEMPTS` fois (la même logique que la file en mémoire), puis le mail part en file morte, intact. Un message illisible ou invalide y part tout de suite, sans réessai.
- **Un arrêt ne perd rien.** Le worker cesse de prendre des mails, laisse les envois en cours finir (`HEROS_MAIL_SEND_TIMEOUT`), puis remet les autres dans la file.
- **L'ordre de démarrage n'a pas d'importance.** L'API et le worker déclarent tous deux les échanges et les files à chaque connexion. Les mails déposés avant le premier worker attendent dans `mail.send`.

### Quand RabbitMQ est bloqué ou absent

L'API ne dépend pas du courtier pour répondre :

- **Courtier injoignable** : l'API démarre quand même, se reconnecte toute seule, et `Send` retourne tout de suite `mail.ErrQueueFull` (aucune requête n'attend). Les appelants journalisent l'échec et continuent, comme avant (l'utilisateur peut redemander son lien d'activation). Un envoi groupé (newsletter) doit réessayer sur `ErrQueueFull`, et c'est ce que le contrat de cette erreur dit.
- **Alarme de ressources** : sous 50 Mo de disque libre (`disk_free_limit`) ou au-delà de la limite de mémoire, RabbitMQ **bloque les dépôts** (« *Publishers will be blocked until this alarm clears* » dans ses journaux). RabbitMQ ne prévient (`connection.blocked`) une connexion qu'au moment où elle publie : le premier dépôt après le début de l'alarme attend `HEROS_RABBITMQ_CONFIRM_TIMEOUT` (5 s) avant d'échouer, et ce mail peut tout de même être accepté à la levée de l'alarme (d'où un doublon possible si l'appelant réessaie). Les dépôts suivants sont refusés aussitôt, avec la même erreur, jusqu'à la levée. Le worker, lui, continue de vider la file. En développement, la cause est presque toujours le disque de la machine virtuelle de Docker : `docker system df`, puis `docker system prune`.
- **Worker absent** : les mails s'accumulent dans `mail.send`, sans rien perdre, et partent au retour du worker. `mailworker -healthcheck` (le contrôle de santé du conteneur) échoue tant qu'aucun worker n'est attaché à la file.

### Exploiter la file morte

L'interface de gestion (<http://localhost:15672>, `heros` / `heros` en développement) montre les files `mail.send` et `mail.dead`. Après avoir corrigé la cause (serveur SMTP, adresse refusée) :

1. Ouvrir `mail.dead`, *Get messages* avec « Ack mode » à « Nack message requeue true » : les messages restent là, on les lit seulement.
2. Pour rejouer un mail : le relire avec « Ack message requeue false » et un seul message (il quitte la file morte), puis *Publish message* sur l'échange `mail`, clé de routage `send`, avec le même corps. Éviter *Purge*, qui vide toute la file.
3. Une file morte qui grossit est un incident : le SMTP est tombé ou refuse les adresses.

Ces messages contiennent l'adresse du destinataire et le contenu du mail (données personnelles) : ne pas les copier ailleurs, et vider la file morte une fois traitée.

### Limites connues

- **Livraison « au moins une fois »**, pas « exactement une fois » : un doublon reste possible (voir plus haut).
- **La publication a lieu après le commit en base**, par l'appelant. Si l'API s'arrête entre le commit et `Send`, le mail n'est pas déposé (acceptable pour l'activation ou la réinitialisation : on redemande). Un envoi groupé qui parcourt ses destinataires dans le processus de l'API perdrait aussi les destinataires restants, car le courtier ne connaît que ce qui lui a été déposé. Le garantir exigerait la table `mail_outbox` ou un job par numéro, que cette file n'apporte pas.
- **Un message qui n'atteint aucune file** (file supprimée à la main pendant que l'API tourne) est perdu. Le `Publisher` le détecte (dépôt *mandatory*), journalise une erreur et redéclare tout en se reconnectant ; les mails suivants passent.
- **Un seul nœud RabbitMQ** et des files classiques durables, sans réplication : suffisant à cette échelle. Le disque du courtier est à sauvegarder avec le reste si l'on tient aux mails en attente.
- **Tests** : les tests d'intégration s'exécutent contre `HEROS_TEST_RABBITMQ_URL` (un courtier jetable : chaque test crée ses propres files et les supprime). Sans cette variable, ils sont ignorés en local et échouent en CI.
