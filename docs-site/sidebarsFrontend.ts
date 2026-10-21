import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// Documentation du frontend (frontend/README.md et frontend/docs/*.md), dans l'ordre de lecture :
// démarrer, comprendre l'architecture, écrire du code, puis brancher l'API. Des libellés courts
// remplacent les titres des documents dans la barre latérale.
const sidebars: SidebarsConfig = {
  frontendSidebar: [
    {type: 'doc', id: 'README', label: 'Démarrer'},
    {type: 'doc', id: 'docs/architecture', label: 'Architecture'},
    {type: 'doc', id: 'docs/conventions', label: 'Conventions'},
    {type: 'doc', id: 'docs/ui', label: "Composants d'interface"},
    {type: 'doc', id: 'docs/backend-contract', label: 'Contrat avec le backend'},
  ],
};

export default sidebars;
