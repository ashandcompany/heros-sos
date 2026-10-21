# Chiffrement des données : cascade à clé de 512 bits

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

Le gain de sécurité viendra du contrôle d'accès, de la 2FA (obligatoire pour les modérateurs, prévue pour les héros en version 2) et de la gestion des clés, pas de l'empilement d'algorithmes.
