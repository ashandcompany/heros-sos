import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';
import type * as OpenApiPlugin from 'docusaurus-plugin-openapi-docs';

// Exécuté par Node.js : pas de code navigateur (window, JSX…) dans ce fichier.

const REPO = 'https://github.com/ashandcompany/heros-sos';

const config: Config = {
  title: 'Heros-SOS',
  tagline: "Architecture, base de données et contrat d'API",
  favicon: 'img/favicon.ico',

  future: {
    v4: true, // compatibilité avec la prochaine version majeure de Docusaurus
  },

  // Déploiement sur GitHub Pages (dépôt public) : https://ashandcompany.github.io/heros-sos/
  // À adapter si le site est hébergé ailleurs (le workflow de déploiement reste à décider).
  url: 'https://ashandcompany.github.io',
  baseUrl: '/heros-sos/',
  organizationName: 'ashandcompany',
  projectName: 'heros-sos',
  trailingSlash: false,

  // Un lien cassé fait échouer le build (et donc le job CI) plutôt que de passer inaperçu.
  onBrokenLinks: 'throw',

  markdown: {
    // STACK.md, MVP.md, TABLES.md, stack/ et tables/ vivent dans backend/ et se lisent aussi sur GitHub : ce sont des
    // fichiers Markdown purs. `detect` ne traite en MDX que les .mdx, sans accolades à échapper ailleurs.
    format: 'detect',
    // Les blocs ```mermaid (schémas des tables dans backend/tables/, schémas d'architecture) sont rendus en
    // diagrammes. GitHub les rend aussi : la source reste lisible aux deux endroits.
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },

  i18n: {
    defaultLocale: 'fr',
    locales: ['fr'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          // Les pages d'accueil et d'API sont à la racine du site (voir docs/intro.mdx).
          routeBasePath: '/',
          sidebarPath: './sidebars.ts',
          // Rend les pages générées depuis l'OpenAPI (docs/api/).
          docItemComponent: '@theme/ApiItem',
          editUrl: `${REPO}/tree/develop/docs-site/`,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  plugins: [
    [
      // Source unique : les documents de conception restent dans backend/, le site les lit sur place.
      // Aucune copie, donc aucune dérive entre GitHub et le site.
      '@docusaurus/plugin-content-docs',
      {
        id: 'backend',
        path: '../backend',
        routeBasePath: 'backend',
        include: ['STACK.md', 'MVP.md', 'TABLES.md', 'stack/*.md', 'tables/*.md'],
        sidebarPath: './sidebarsBackend.ts',
        editUrl: `${REPO}/tree/develop/backend/`,
      },
    ],
    [
      // Pages d'API générées depuis backend/openapi/openapi.yaml à chaque `npm run build` / `npm start`
      // (voir les scripts de package.json). docs/api/ n'est jamais versionné.
      'docusaurus-plugin-openapi-docs',
      {
        id: 'api',
        docsPluginId: 'classic',
        config: {
          heros: {
            specPath: '../backend/openapi/openapi.yaml',
            outputDir: 'docs/api',
            sidebarOptions: {
              groupPathsBy: 'tag',
            },
          } satisfies OpenApiPlugin.Options,
        },
      },
    ],
  ],

  themes: ['@docusaurus/theme-mermaid', 'docusaurus-theme-openapi-docs'],

  themeConfig: {
    colorMode: {
      respectPrefersColorScheme: true,
    },
    mermaid: {
      theme: {light: 'neutral', dark: 'dark'},
    },
    navbar: {
      title: 'Heros-SOS',
      logo: {
        alt: 'Heros-SOS',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docSidebar',
          position: 'left',
          label: 'Présentation',
        },
        {
          type: 'docSidebar',
          docsPluginId: 'backend',
          sidebarId: 'backendSidebar',
          position: 'left',
          label: 'Backend',
        },
        {
          type: 'docSidebar',
          sidebarId: 'apiSidebar',
          position: 'left',
          label: 'API',
        },
        {
          href: REPO,
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {label: 'Périmètre du MVP', to: '/backend/MVP'},
            {label: 'Choix techniques', to: '/backend/STACK'},
            {label: 'Tables', to: '/backend/TABLES'},
          ],
        },
        {
          title: 'Projet',
          items: [{label: 'GitHub', href: REPO}],
        },
      ],
      copyright: `Heros-SOS © ${new Date().getFullYear()}. Site construit avec Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'go', 'sql', 'yaml', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
