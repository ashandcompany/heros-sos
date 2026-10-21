# MVP de Heros-SOS

Le MVP permet de jouer le cœur du projet : un citoyen signale un incident, un héros le prend en charge et le clôture, un modérateur surveille (cahier §1.3).

## Scénario de démo

1. Un citoyen s'inscrit, active son compte par email, se connecte.
2. Il consulte les incidents en cours pour éviter un doublon, puis déclare un incident.
3. Un modérateur valide un compte héros, et peut supprimer un doublon ou changer la priorité.
4. Le héros reçoit l'alerte, voit la liste par priorité, prend l'incident, puis le clôture (réussi ou échoué).
5. Les incidents réussis apparaissent dans le carrousel de la page d'accueil.

## Dans le MVP

| Domaine | Contenu |
|---|---|
| **Comptes** | Inscription (§5.4), activation par email, connexion, déconnexion, `/auth/me`, mot de passe oublié, sessions |
| **Rôles** | Citoyen, héros, modérateur ; demande « Devenir super-héros » validée par un modérateur |
| **Incidents** | Déclaration, liste des incidents en cours, mes incidents, détail, priorité, suppression d'un doublon |
| **Interventions** | Un héros prend un incident et le clôture. Un seul héros par intervention : la table `intervention_heroes` est déjà prête pour l'équipe, rien ne sera à refaire. |
| **Public** | Page d'accueil (derniers incidents réussis), liste des héros (alias, photo, description), liste des vilains |
| **Modération** | Valider les héros, gérer les incidents, désactiver un compte, gérer les vilains |
| **Mails** | Activation, mot de passe oublié, nouvel incident vers héros et modérateurs |
| **Sécurité de base** | argon2id, cookie de session, CSRF, limitation de débit sur la connexion, contrôle d'accès à chaque endpoint, validation des entrées, TLS, journal d'audit simple |

### Tables concernées (10 sur 25)

`users`, `hero_profiles`, `sessions`, `user_tokens`, `incident_types`, `incidents`, `interventions`, `intervention_heroes`, `villains`, `audit_logs`.

Ajouter `villain_specialties` et `incident_villains` si on lie les vilains aux incidents.

## Hors MVP (version 2)

- Notes et réponses des héros, films et OMDb, statistiques (§5.8.1 et §5.8.2).
- Newsletter et double opt-in, articles du blog.
- Équipes de héros : invitations, confirmations, validation du résultat, arbitrage par un modérateur.
- 2FA TOTP, liste des appareils connectés, changement d'email modéré.
- Proximité des héros (`hero_locations`), encyclopédie Marvel.

## À ne pas couper

- **`internal/crypto` (cascade AES-256-GCM puis XChaCha20-Poly1305) dès le départ.** Il touche tous les accès à `users`. L'ajouter après obligerait à migrer des données déjà enregistrées en clair. Exception : un MVP qui reste une démo avec des données fictives.
- **La 2FA des modérateurs avant tout vrai utilisateur.** Un compte modérateur piraté donne accès à tout. Elle peut attendre pour une démo, pas pour une mise en ligne.

## Mails et RabbitMQ

Pour le MVP, les mails partent depuis le worker ou une goroutine, derrière une interface `Mailer`. RabbitMQ s'y branche ensuite sans changer le reste. L'intégrer dès le MVP est possible, mais cela ajoute un service à héberger et à tester avant le premier incident déclaré.

## Ordre de construction

1. **Socle :** `go mod init`, Docker, `openapi.yaml`, `/health`, intégration continue.
2. **Base de données :** migrations, modèles GORM, `internal/crypto`.
3. **Authentification de bout en bout**, branchée sur les pages d'Emma (inscription, connexion, mot de passe oublié).
4. **Incidents et interventions.**
5. **Modération et pages publiques.**
6. **Mails**, puis finitions et déploiement.

## À vérifier avec Emma

Les pages de connexion existent côté front avec des données mockées. À confirmer : la déclaration d'incident, l'espace héros et le dashboard modérateur existent-ils déjà ?

Voir aussi [STACK.md](STACK.md) pour les choix techniques et [TABLES.md](TABLES.md) pour le schéma.
