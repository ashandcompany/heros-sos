# Champs chiffrés

| Table | Colonnes 🔒 |
|---|---|
| `users` | `last_name`, `first_name`, `birth_date`, `phone`, `street`, `address_complement`, `postal_code`, `city`, `country` |
| `user_mfa` | `totp_secret` |

Restent en clair : `users.email` (connexion), `hero_profiles.alias` (public), le lieu des incidents (détection des doublons), la position actuelle des héros (`hero_locations`, si la proximité est retenue), tout ce qui doit être filtré ou trié en SQL. Chaque champ chiffré porte son propre nonce et ses tags (environ 68 octets de plus).
