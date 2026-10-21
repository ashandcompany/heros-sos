# AES, RSA et protocoles

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
