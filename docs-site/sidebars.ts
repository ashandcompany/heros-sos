import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';
// Généré par `npm run api:gen` depuis backend/openapi/openapi.yaml (non versionné) : exporte la liste
// des pages d'API, groupées par tag.
import apiSidebar from './docs/api/sidebar';

const sidebars: SidebarsConfig = {
  docSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Architecture',
      link: {type: 'doc', id: 'architecture'},
      items: ['architecture/connexion'],
    },
    'fonctionnement',
  ],
  apiSidebar,
};

export default sidebars;
