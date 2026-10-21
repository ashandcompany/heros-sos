# Schéma complet

Toutes les relations des schémas par domaine en un seul. Dense : utile pour retrouver une relation précise, pas pour comprendre la structure.

Légende des cardinalités : voir la [vue d'ensemble](../TABLES.md#légende-des-schémas).

```mermaid
erDiagram
    USERS ||--o| HERO_PROFILES : "peut être"
    USERS ||--o{ SESSIONS : "ouvre"
    USERS ||--o{ USER_TOKENS : "reçoit"
    USERS ||--o| USER_MFA : "protège"
    USERS ||--o{ EMAIL_CHANGE_REQUESTS : "demande"
    USERS ||--o{ INCIDENTS : "déclare"
    HERO_PROFILES ||--o{ INTERVENTION_HEROES : "participe"
    HERO_PROFILES ||--o{ HERO_FILMS : "référence"
    HERO_PROFILES ||--o{ HERO_REVIEW_REPLIES : "répond"
    HERO_PROFILES ||--o| HERO_LOCATIONS : "partage sa position"
    USERS ||--o{ HERO_REVIEWS : "note (citoyen)"
    USERS ||--o{ ARTICLES : "rédige"
    USERS ||--o{ NEWSLETTERS : "envoie"
    USERS ||--o{ AUDIT_LOGS : "agit"
    USERS |o--o{ NEWSLETTER_SUBSCRIPTIONS : "s'abonne"
    INCIDENT_TYPES ||--o{ INCIDENTS : "classe"
    INCIDENT_TYPES ||--o{ VILLAIN_SPECIALTIES : "spécialité"
    VILLAINS ||--o{ VILLAIN_SPECIALTIES : "pratique"
    INCIDENTS ||--o{ INTERVENTIONS : "reçoit"
    INTERVENTIONS ||--|{ INTERVENTION_HEROES : "réunit"
    INCIDENTS ||--o{ INCIDENT_VILLAINS : "implique"
    VILLAINS ||--o{ INCIDENT_VILLAINS : "impliqué dans"
    INTERVENTION_HEROES ||--o| HERO_REVIEWS : "est noté"
    HERO_REVIEWS ||--o| HERO_REVIEW_REPLIES : "reçoit"
    FILMS ||--o{ HERO_FILMS : "apparaît dans"
    FILMS ||--o{ VILLAIN_FILMS : "met en scène"
    VILLAINS ||--o{ VILLAIN_FILMS : "apparaît dans"
```
