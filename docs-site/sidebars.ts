import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';
// Généré par `npm run api:gen` depuis backend/openapi/openapi.yaml (non versionné) : exporte la liste
// des pages d'API, groupées par tag.
import apiSidebar from './docs/api/sidebar';

const sidebars: SidebarsConfig = {
  docSidebar: ['intro', 'architecture'],
  apiSidebar,
};

export default sidebars;
