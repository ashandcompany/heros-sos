import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// Documents de conception du backend (backend/*.md), dans l'ordre de lecture : ce que l'on construit,
// avec quoi, puis le schéma de base de données. La stack (backend/stack/) et les tables (backend/tables/)
// sont découpées en pages ; STACK.md et TABLES.md en sont les pages d'accueil, ouvertes en cliquant sur
// la catégorie.
const sidebars: SidebarsConfig = {
  backendSidebar: [
    'MVP',
    {
      type: 'category',
      label: 'Stack du backend',
      link: {type: 'doc', id: 'STACK'},
      items: [
        {type: 'doc', id: 'stack/base-de-donnees', label: 'Base de données'},
        {type: 'doc', id: 'stack/openapi', label: 'OpenAPI / Swagger'},
        {type: 'doc', id: 'stack/docker', label: 'Docker'},
        {type: 'doc', id: 'stack/rabbitmq', label: 'RabbitMQ'},
        {type: 'doc', id: 'stack/performance-et-synchronisation', label: 'Performance et synchronisation'},
        {type: 'doc', id: 'stack/elk', label: 'Monitoring ELK'},
        {type: 'doc', id: 'stack/omdb', label: 'OMDb pour les films'},
        {type: 'doc', id: 'stack/securite', label: 'Sécurité et chiffrement'},
        {type: 'doc', id: 'stack/chiffrement', label: 'Chiffrement des champs'},
        {type: 'doc', id: 'stack/sessions', label: 'Sessions'},
        {type: 'doc', id: 'stack/aes-rsa', label: 'AES, RSA et protocoles'},
      ],
    },
    {
      type: 'category',
      label: 'Tables de la base de données',
      link: {type: 'doc', id: 'TABLES'},
      items: [
        {type: 'doc', id: 'tables/comptes', label: '1. Comptes et accès'},
        {type: 'doc', id: 'tables/incidents', label: '2. Incidents'},
        {type: 'doc', id: 'tables/vilains', label: '3. Super-héros et super-vilains'},
        {type: 'doc', id: 'tables/films', label: '4. Films des héros et des vilains'},
        {type: 'doc', id: 'tables/reputation', label: '5. Réputation'},
        {type: 'doc', id: 'tables/contenu', label: '6. Contenu et newsletter'},
        {type: 'doc', id: 'tables/technique', label: '7. Technique'},
        {
          type: 'category',
          label: 'Annexes',
          collapsed: true,
          items: [
            {type: 'doc', id: 'tables/schema-complet', label: 'Schéma complet'},
            {type: 'doc', id: 'tables/champs-chiffres', label: 'Champs chiffrés'},
            {type: 'doc', id: 'tables/statistiques', label: 'Statistiques'},
            {type: 'doc', id: 'tables/points-ouverts', label: 'Points ouverts'},
          ],
        },
      ],
    },
  ],
};

export default sidebars;
