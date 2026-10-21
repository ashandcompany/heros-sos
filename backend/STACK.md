# Stack du backend

Complément au cahier des charges, qui prévoit PHP 8.4 et MySQL (§2.4) : le backend est en **Go** avec **PostgreSQL**. Le frontend est en **Next.js** (dossier `frontend/`) et parle à l'API Go en REST. Le schéma des tables est dans [TABLES.md](TABLES.md).

## Base de données

PostgreSQL remplace MySQL, accédé avec **GORM** (`gorm.io/gorm` et `gorm.io/driver/postgres`, basé sur `pgx`).

- **Migrations :** fichiers SQL versionnés avec `golang-migrate`. `AutoMigrate` de GORM n'est utilisé qu'en développement, jamais en production (il ne supprime rien et ne se relit pas).
- **Connexion :** TLS, utilisateur applicatif à droits minimaux (pas de superutilisateur).
- **Identifiants :** UUID générés côté Go (v7), pour que l'identifiant de la ligne soit connu avant l'insertion et serve de donnée associée (AAD) au chiffrement des champs.
- **Modèles :** les structs GORM ne sont jamais renvoyées par l'API. Les types générés par OpenAPI (DTO) sont séparés des entités, pour ne pas exposer le hash du mot de passe ou des champs chiffrés.
- **Suppression RGPD :** pas de `gorm.DeletedAt` (suppression logique) sur les données personnelles, car la ligne resterait en base. Suppression réelle (`Unscoped().Delete`) ou anonymisation, pour le droit à l'effacement.
- **Chiffrement des champs :** fait dans le repository ou dans les hooks `BeforeSave` et `AfterFind`, avec `internal/crypto`. Un type `Valuer/Scanner` seul ne connaît pas l'identifiant de la ligne, donc il ne convient pas pour l'AAD.
- **Injection SQL :** GORM paramètre ses requêtes. Le risque vient de `Raw`, `Exec` et `Where` avec une chaîne concaténée : toujours des `?` avec des arguments.
- **Mails (repli outbox, sans RabbitMQ) :** `clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}`.
- **Les sessions, les mails à envoyer et les champs chiffrés** (voir plus bas) vivent dans cette même base.

Compromis par rapport à `sqlc` : GORM va plus vite à écrire, mais le SQL n'est pas vérifié à la compilation, et `Preload` peut provoquer des requêtes en trop. Rien de bloquant à cette échelle.

## OpenAPI / Swagger

Deux approches étaient possibles. **Décision : spec d'abord** (`openapi.yaml` et `oapi-codegen`), et le front peut générer son client TypeScript depuis le même fichier.

| Approche                       | Outil                                                                                              | Avantage                                                                                             | Compromis                                                     |
| ------------------------------ | -------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| **Spec d'abord** (recommandée) | Fichier `openapi.yaml` (OpenAPI 3) et `oapi-codegen` pour générer les types et l'interface serveur | Le contrat est la référence, le front peut générer son client, le code ne peut pas dériver de la doc | Il faut écrire le YAML d'abord                                |
| **Code d'abord**               | `swaggo/swag` (annotations en commentaires)                                                        | Rapide à démarrer                                                                                    | Génère surtout du Swagger 2.0, et la doc peut dériver du code |

Dans les deux cas, Swagger UI est servi sur `/docs`, uniquement hors production. Les routes sont préfixées par `/api/v1`, valeur déjà prévue par le README du frontend. Les noms de rôles de l'API (`citizen`, `hero`, `moderator`) devront être alignés avec ceux du frontend (`citoyen`, `super-heros`).

## Docker

Un `docker-compose.yml` avec ces services :

- `api` : image Go en multi-stage, avec une image finale minimale.
- `worker` : worker Go qui lit la file `mail.send` et envoie les mails.
- `proxy` : Caddy ou Traefik. Il termine le TLS et route `/api/*` vers `api` et le reste vers Next.js, pour que le navigateur ne parle qu'à une seule origine (voir « Sessions »).
- `web` : Next.js, si le frontend est aussi conteneurisé (à confirmer avec Emma).
- `db` : PostgreSQL, avec un volume.
- `rabbitmq` : retenu pour les mails, avec l'interface de management (à condition que l'hébergeur supporte Docker).
- `mailpit` : faux serveur SMTP pour voir les mails en développement.
- `adminer` ou pgAdmin dans un profil `dev` uniquement : le cahier §2.4 autorise phpMyAdmin seulement en développement et interdit tout outil équivalent en production (RGPD).

Les secrets passent par `.env`, avec un `.env.example` commité et jamais le vrai fichier.

## RabbitMQ : utile ou pas ?

**Oui, mais seulement pour les mails.** Le cahier §4.3 déclenche des envois groupés : un incident prévient tous les modérateurs et tous les héros, et la newsletter part vers tous les utilisateurs. Sans file, la requête HTTP attend le SMTP. Elle échoue ou expire si le serveur de mail est lent, et un mail raté n'est pas rejoué.

Proposition :

- une file `mail.send` avec un worker Go séparé, qui retente en cas d'échec et envoie les échecs définitifs dans une dead-letter queue ;
- aucune autre file pour l'instant. Les statistiques se calculent à la demande en SQL.

Deux points à connaître :

- **Hébergement :** le cahier §2.1 dit que chaque groupe héberge chez son hébergeur. Un hébergement mutualisé ne fera pas tourner RabbitMQ. Vérifier que l'hébergeur supporte Docker.
- **Alternative sans RabbitMQ :** une table `mail_outbox` en PostgreSQL lue par un worker (`SELECT … FOR UPDATE SKIP LOCKED` évite que deux workers prennent le même mail). Il n'y a rien de plus à héberger et c'est suffisant à cette échelle. RabbitMQ est un peu surdimensionné ici, mais il se justifie pour la démonstration technique ou si d'autres événements sont prévus.

Avec RabbitMQ, le message est publié après le commit en base, avec confirmation de publication. Sinon un crash entre les deux ferait perdre un mail.

**Décision :** RabbitMQ est retenu pour les mails, tant que l'hébergeur supporte Docker. Sinon, repli sur la table `mail_outbox` (voir [TABLES.md](TABLES.md)).

## OMDb pour les films

[OMDb](https://omdbapi.com/) remplace « l'API IMDb » du cahier, qui est payante. Il couvre :

- `?s=titre` pour chercher,
- `?i=tt…` pour le détail par identifiant IMDb,
- `?t=titre` pour un titre exact,
- une réponse en JSON.

Le quota gratuit par jour n'est pas indiqué sur le site : à regarder au moment de demander la clé, qui arrive par email. Autres points :

- **Licence :** le contenu est sous CC BY-NC 4.0 (non commercial), ce qui est acceptable pour un projet pédagogique.
- **Posters :** l'API d'images dédiée est réservée aux patrons.
- **Clé :** elle reste côté backend, dans `.env`, jamais envoyée au front.
- **Cache :** le film est stocké en base à l'ajout, ce qui économise le quota. De toute façon un modérateur doit valider chaque film (§5.8.4).

OMDb ne remplace pas l'encyclopédie Marvel du §6.1.4, qui reste à trancher.

## Sécurité et chiffrement : ce qui est retenu

Le RGPD n'impose pas d'algorithme précis (article 32 : mesures appropriées, état de l'art). Le chiffrement symétrique est déjà résistant au quantique : AES-256 garde environ 128 bits de sécurité face à Grover. Le post-quantique ne compte que là où l'on utilise de la clé publique.

| Mesure | Pertinence | Où dans le projet |
|---|---|---|
| **Cascade AES-256-GCM puis XChaCha20-Poly1305** (clé de 512 bits) sur les données personnelles | Oui, retenu : 512 bits de clé réelle avec des algorithmes standards | Champs sensibles en base, pour tous les utilisateurs : nom, prénom, date de naissance, téléphone et adresse postale (§5.4 et §5.7). Un paquet `internal/crypto` avec un format versionné pour pouvoir changer d'algorithme plus tard. Détail ci-dessous. |
| **TLS avec échange hybride ML-KEM** (`X25519MLKEM768`) | Oui, gratuit avec une version récente de Go et un navigateur à jour. Protège contre « récolter maintenant, déchiffrer plus tard ». | Navigateur vers Next.js, Next.js vers l'API Go, API vers la base et vers le SMTP. Le composant qui termine le TLS (proxy Docker ou Go) doit être assez récent : à vérifier. |
| **Partage de secret de Shamir** pour la clé maîtresse | Oui, simple et utile | Opérations, pas le code applicatif : la clé maîtresse qui protège les clés de données et les sauvegardes est répartie entre *n* personnes de l'équipe, *k* suffisent pour la reconstituer. |
| **Ed25519** pour des liens signés | Oui, car ils sont de courte durée et le risque quantique est négligeable | Lien de désabonnement de la newsletter (§4.2), lien de notation d'un héros dans le mail de fin d'intervention (§4.3.3 et §6.1.3). Aussi les clés SSH de déploiement. Un identifiant `kid` permet de renouveler la clé. |
| ML-DSA, FHE, MPC, QKD, Threefish | Non, hors de proportion pour ce projet | Non utilisés. Threefish pourra s'ajouter plus tard comme autre id d'algorithme du format versionné. |

### Chiffrement des données : cascade à clé de 512 bits

**Objectif :** que la base de données seule ne révèle aucune donnée personnelle. Si la base, une sauvegarde, un disque ou une lecture SQL directe fuite sans la clé maîtresse, les champs sensibles restent illisibles. C'est la mesure de chiffrement de l'article 32 du RGPD, et des données chiffrées avec les règles de l'art peuvent éviter d'avoir à prévenir chaque personne concernée en cas de fuite (article 34).

- **Protège contre :** vol d'un dump ou d'une sauvegarde, lecture directe de la base (injection SQL qui lit les colonnes, hébergeur ou administrateur de base curieux), disque volé.
- **Ne protège pas contre :** un contrôle d'accès défaillant dans l'API (elle déchiffre pour la personne qui demande), une session volée, un serveur applicatif compromis où les clés sont en mémoire. Là, ce sont le contrôle d'accès et la 2FA qui comptent.
- **Pourquoi une cascade à 512 bits :** le niveau de sécurité d'AES-256 suffit déjà. La cascade répond à l'exigence d'une clé de 512 bits réelle, et ajoute une défense en profondeur : une faille dans l'un des deux algorithmes ou dans son implémentation n'expose pas la donnée.
- **Ne sert pas pour :** les sessions (jeton aléatoire haché), les mots de passe (argon2id), l'email de connexion et le lieu des incidents.

AES n'existe pas en 512 bits. On obtient 512 bits de clé réelle en chaînant deux algorithmes standards avec deux clés indépendantes :

- **Clé maîtresse :** 64 octets tirés de `crypto/rand`, protégés par Shamir et hors du dépôt.
- **Sous-clés :** HKDF-SHA-512 dérive deux clés de 32 octets avec des libellés distincts (une pour AES-256-GCM, une pour XChaCha20-Poly1305).
- **Chiffrement :** d'abord AES-256-GCM (nonce de 12 octets aléatoire), puis XChaCha20-Poly1305 sur le résultat (nonce de 24 octets aléatoire). Le déchiffrement se fait dans l'ordre inverse.
- **Données associées (AAD) :** table, colonne, identifiant de l'enregistrement et version de clé, pour qu'un chiffré ne puisse pas être déplacé vers une autre ligne.
- **Format stocké :** `[version | id algo | id clé | nonce 1 | nonce 2 | chiffré avec ses deux tags]`, avec l'id algo `aes256gcm+xchacha20poly1305`. Un autre algorithme, comme Threefish ou un hybride post-quantique, pourra s'ajouter plus tard avec un autre id.
- **Bibliothèques :** `crypto/aes` et `crypto/cipher` (standard), `golang.org/x/crypto/chacha20poly1305` (`NewX`), HKDF.
- **Coût :** environ 68 octets de plus par champ (deux nonces, deux tags) et deux passes de chiffrement, négligeable pour des champs courts.
- **Renouvellement :** nouvelle version de clé avant 2³² messages avec la même clé (limite du nonce aléatoire de GCM), largement hors d'atteinte à cette échelle.

Si l'un des deux algorithmes est un jour affaibli, l'autre protège encore la donnée. Les clés doivent rester indépendantes : jamais la même clé pour les deux.

À retenir :

- **Mots de passe :** hachés avec argon2id, jamais chiffrés. L'activation de compte et le mot de passe oublié (§4.3.1, §4.3.4) utilisent un token aléatoire stocké haché en base.
- **Champs chiffrés :** ils ne sont plus filtrables ni triables en SQL. On ne chiffre pas ce qu'il faut chercher : l'email de connexion et le lieu de l'incident (détection des doublons, §4.1.1) restent exploitables.
- **Clés :** générées avec `crypto/rand`, hors du dépôt (`.env` ou gestionnaire de secrets), renouvelables.

Le gain de sécurité viendra du contrôle d'accès, de la 2FA (obligatoire pour modérateurs et héros) et de la gestion des clés, pas de l'empilement d'algorithmes.

## Sessions

Le cahier §2.2 impose une gestion des sessions via le formulaire de connexion : sessions côté serveur, pas de JWT. Elles se révoquent immédiatement, par exemple quand un modérateur désactive un compte (§3.1.3).

**Jeton**

- 32 octets tirés de `crypto/rand`, encodés en base64url. C'est un identifiant opaque, sans donnée.
- En base, seul le hash SHA-256 du jeton est stocké. Le jeton brut n'existe que dans le cookie : une fuite de la base ne donne aucune session utilisable.
- Table `sessions` : `token_hash`, `user_id`, `created_at`, `last_seen_at`, `expires_at`, `mfa_verified_at`, `revoked_at`, IP et résumé de l'appareil `device_label` (« Chrome · macOS », calculé à partir du User-Agent dont la chaîne brute n'est pas conservée ; données personnelles, durée de conservation limitée).
- Stockage dans la base principale, pas dans Redis : un service de moins à héberger (§2.1), révocation immédiate, une lecture par requête négligeable à cette échelle.

**Cookie** : `__Host-session`, `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`, sans attribut `Domain`.

**Next.js et API Go en même origine** : un reverse proxy (Caddy ou Traefik, dans Docker) route `/api/*` vers Go et le reste vers Next.js. Pas de CORS, et le cookie fonctionne sans configuration spéciale. Le middleware de Next ne sert qu'à rediriger : l'autorisation se vérifie toujours dans l'API Go. Un endpoint `/api/me` renvoie le rôle, utilisé pour le §5.3 (redirection d'un citoyen vers son profil, d'un héros vers son espace pro).

**Du cookie au navigateur (option A : Next relaie le cookie de Go)**

La session vit dans le backend Go : il la crée, la stocke, la valide et la révoque. Le front ne fait que porter le cookie, et ne décide jamais d'un droit (il redirige pour l'ergonomie, chaque endpoint Go revérifie).

- Go crée la session et renvoie `Set-Cookie: __Host-session=…` avec l'expiration. Les attributs du cookie sont définis ici, une seule fois.
- Les Server Actions de Next (connexion, inscription, validation du code 2FA) appellent Go côté serveur, lisent le `Set-Cookie` de la réponse (`getSetCookie()`) et le réappliquent au navigateur avec `cookies().set(...)`.
- Pour les appels suivants, Next relaie l'en-tête `Cookie` reçu à Go. Go lit uniquement le cookie, jamais un en-tête `Authorization`.
- Le jeton n'est jamais lisible en JavaScript (`HttpOnly`).

Endpoints d'authentification (préfixe `/api/v1`, noms définitifs dans `openapi.yaml`) :

| Endpoint | Rôle |
|---|---|
| `POST /auth/register` | Inscription, envoie l'email d'activation |
| `POST /auth/activate` | Active le compte avec le token du lien |
| `POST /auth/login` | Connexion par email, ou par alias pour un héros. Répond `ok` ou `mfa_required`. |
| `POST /auth/mfa/verify` | Valide le code TOTP et ouvre la session complète |
| `POST /auth/logout`, `POST /auth/logout-all` | Déconnexion de l'appareil, ou de tous les appareils |
| `GET /auth/me` | Utilisateur et rôle de la session (redirections du §5.3) |
| `POST /auth/password-reset`, `POST /auth/password-reset/confirm` | Demande, puis changement avec le token reçu |
| `GET /auth/sessions`, `DELETE /auth/sessions/{id}` | Liste des appareils connectés et révocation |
| `POST /heroes/requests` | Demande du statut de super-héros (formulaire « Devenir super-héros ») |

À adapter dans le front d'Emma : nom du cookie `__Host-session` en production, expiration reçue de l'API au lieu des 7 jours fixes, relais de l'en-tête `Cookie` à la place du `Bearer`, état « code 2FA requis » dans le résultat de la connexion, et remplacement des jetons `mock-token-<id>` (prévisibles) par le vrai jeton aléatoire.

**Durées** (à ajuster)

| Profil         | Inactivité | Durée maximale                                 |
| -------------- | ---------- | ---------------------------------------------- |
| Citoyen, héros | 2 h        | 7 jours (30 jours avec « se souvenir de moi ») |
| Modérateur     | 15 min     | 8 h                                            |

**Règles**

- Nouveau jeton à la connexion, à la validation de la 2FA et au changement de mot de passe (anti-fixation).
- Mot de passe changé ou réinitialisé : toutes les autres sessions de l'utilisateur sont révoquées.
- Compte désactivé par un modérateur : ses sessions sont supprimées immédiatement.
- Compte non activé (§4.3.1) : pas de session. Héros non vérifié : rôle citoyen limité jusqu'à validation.
- CSRF : vérification de l'en-tête `Origin` et jeton CSRF sur toutes les requêtes qui modifient des données.
- 2FA (TOTP) : après le mot de passe, la session est « en attente de 2FA » et ne donne accès à rien. `mfa_verified_at` est renseigné une fois le code validé. Les actions sensibles (valider un héros, envoyer la newsletter, supprimer un incident) exigent une 2FA récente.
- Connexion : limitation de débit, délai croissant après échecs, message d'erreur identique que le compte existe ou non.
- « Mon compte » liste les appareils connectés avec un bouton de révocation et « se déconnecter partout ».
- Une tâche supprime les sessions expirées.

## Décisions prises

- **OpenAPI :** spec d'abord (`oapi-codegen`).
- **Cookie de session :** option A, Next relaie le `Set-Cookie` de Go.
- **Devenir héros :** formulaire dédié « Devenir super-héros » dans « Mon compte », validé par un modérateur.
- **2FA :** TOTP.
- **Newsletter :** double opt-in pour les visiteurs anonymes.
- **Appareil de la session :** résumé (`device_label`), pas de User-Agent brut.
- **Chiffrement :** nom, prénom, date de naissance, téléphone et adresse de tous les utilisateurs.
- **« Citoyen vérifié » :** information de confiance, sans droit supplémentaire.
- **RabbitMQ** pour les mails, si l'hébergeur supporte Docker.

## Décisions en attente

1. **Proximité des héros :** à décider avec le client. Si elle est retenue : la position actuelle du héros (là où il se trouve, pas son domicile), partagée volontairement et visible des modérateurs, en clair, sans historique. Voir `hero_locations` dans [TABLES.md](TABLES.md).
2. **Encyclopédie Marvel** (§6.1.4) : source à vérifier, l'API Marvel semble fermée.

## Prochaines étapes

- Mettre à jour la section Stack du `README.md` (Go, PostgreSQL, Docker, OpenAPI, OMDb, plus RabbitMQ).
- Initialiser `backend/` : `go mod init`, structure, `Dockerfile`, `docker-compose.yml` et `openapi.yaml`, avec `/health` comme premier endpoint.

AES et RSA ne sont pas des protocoles en eux-mêmes : ce sont des **algorithmes**. Voici leurs variantes, leurs modes d'utilisation et les protocoles qui les emploient.

## AES (chiffrement symétrique)

**Tailles de clé**

- AES-128 : 10 tours
- AES-192 : 12 tours
- AES-256 : 14 tours

Dans tous les cas, le bloc fait 128 bits.

**Modes de chiffrement classiques (confidentialité seule)**

- **ECB** (Electronic Codebook) : à éviter, car il révèle les motifs du texte clair.
- **CBC** (Cipher Block Chaining) : courant, nécessite un IV aléatoire.
- **CFB** (Cipher Feedback)
- **OFB** (Output Feedback)
- **CTR** (Counter) : transforme AES en chiffrement par flot.
- **XTS** : conçu pour le chiffrement de disque (BitLocker, LUKS, FileVault).

**Modes authentifiés (AEAD : confidentialité et intégrité)**

- **GCM** (Galois/Counter Mode) : le plus utilisé aujourd'hui (TLS, IPsec).
- **CCM** (Counter with CBC-MAC) : Wi-Fi WPA2, Bluetooth, IoT.
- **GCM-SIV** : résistant à la réutilisation de nonce.
- **OCB** : très rapide.
- **EAX**
- **SIV** (Synthetic IV)

**Autres usages**

- **AES Key Wrap** (KW / KWP, RFC 3394 et 5649) : chiffrement de clés.
- **CMAC** (AES-CMAC) : code d'authentification de message.
- **GMAC** : authentification seule, variante de GCM.

## RSA (chiffrement asymétrique)

**Tailles de clé**

- 1024 bits : obsolète.
- 2048 bits : minimum actuel.
- 3072 bits et 4096 bits : recommandés pour le long terme.

**Schémas de chiffrement**

- **RSAES-PKCS1-v1_5** : ancien, vulnérable à l'attaque de Bleichenbacher.
- **RSAES-OAEP** (Optimal Asymmetric Encryption Padding) : recommandé.
- **RSA-KEM** : encapsulation de clé.
- **Textbook RSA** (sans padding) : jamais sûr en pratique.

**Schémas de signature**

- **RSASSA-PKCS1-v1_5** : encore très répandu (certificats).
- **RSASSA-PSS** (Probabilistic Signature Scheme) : recommandé, obligatoire dans TLS 1.3.

**Échange de clé**

- **Transport de clé RSA** : TLS ≤ 1.2, supprimé de TLS 1.3 faute de confidentialité persistante (forward secrecy).

## Protocoles qui utilisent AES et/ou RSA

| Protocole                                              | AES                                            | RSA                                                   |
| ------------------------------------------------------ | ---------------------------------------------- | ----------------------------------------------------- |
| **TLS / HTTPS**                                        | AES-GCM, AES-CCM, AES-CBC (anciennes versions) | Signatures de certificats, échange de clé (≤ TLS 1.2) |
| **SSH**                                                | aes128/256-ctr, aes-gcm                        | Clés d'hôte et d'utilisateur (rsa-sha2-256/512)       |
| **IPsec / IKE**                                        | AES-CBC, AES-GCM, AES-CTR                      | Authentification par certificats                      |
| **WPA2 / WPA3** (Wi-Fi)                                | AES-CCMP, AES-GCMP                             | via EAP-TLS en entreprise                             |
| **OpenVPN / WireGuard**                                | OpenVPN : AES-GCM (WireGuard utilise ChaCha20) | OpenVPN : certificats RSA                             |
| **PGP / GPG, S/MIME**                                  | Chiffrement du message                         | Chiffrement de la clé de session et signatures        |
| **Kerberos**                                           | aes128/256-cts-hmac-sha1/sha2                  | —                                                     |
| **Signal / messageries**                               | AES-CBC / AES-GCM                              | — (courbes elliptiques)                               |
| **Chiffrement de disque** (BitLocker, LUKS, FileVault) | AES-XTS                                        | —                                                     |
| **PKI / X.509**                                        | —                                              | Signatures de certificats                             |
| **JWT / JOSE**                                         | A128GCM, A256GCM, A256KW                       | RS256, PS256, RSA-OAEP                                |

Je peux détailler le fonctionnement d'un mode en particulier (par exemple GCM ou OAEP), ou te faire une fiche de révision si c'est pour un cours.
