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
