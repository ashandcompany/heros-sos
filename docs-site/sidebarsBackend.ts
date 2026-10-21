import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// Documents de conception du backend (backend/*.md), dans l'ordre de lecture : ce que l'on construit,
// avec quoi, puis le schéma de base de données.
const sidebars: SidebarsConfig = {
  backendSidebar: ['MVP', 'STACK', 'TABLES'],
};

export default sidebars;
