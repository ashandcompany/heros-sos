// Plugin remark : envoie vers GitHub les liens relatifs qui sortent du site.
//
// Les documents du frontend renvoient au code (`../src/lib/api/http.ts`), à l'OpenAPI ou à des fichiers
// Markdown non publiés (`../AGENTS.md`). Sur GitHub ces liens marchent ; sur le site ils mèneraient à une
// page inexistante (et `onBrokenLinks: 'throw'` ferait échouer le build). Ce plugin les remplace par le
// fichier sur GitHub, et laisse intacts les liens vers une page du site, que Docusaurus résout lui-même.

import path from 'node:path';

type Node = {
  type: string;
  url?: string;
  children?: Node[];
};

export type RepoLinksOptions = {
  /** Racine du dépôt sur le disque. */
  repoRoot: string;
  /** Adresse des fichiers sur GitHub, ex : https://github.com/org/repo/blob/develop */
  blobUrl: string;
  /** Vrai si le fichier (chemin absolu) est une page du site. */
  isPage: (file: string) => boolean;
};

// Lien absolu (http:, mailto:…), racine du site (/backend/…) ou ancre de la page : rien à faire.
const NOT_RELATIVE = /^([a-z][a-z0-9+.-]*:|\/|#)/i;

export default function repoLinks({repoRoot, blobUrl, isPage}: RepoLinksOptions) {
  return (tree: Node, file: {path: string}) => {
    const visit = (node: Node) => {
      if ((node.type === 'link' || node.type === 'definition') && node.url && !NOT_RELATIVE.test(node.url)) {
        const [target, hash] = node.url.split('#', 2);
        const absolute = path.resolve(path.dirname(file.path), decodeURI(target));
        const relative = path.relative(repoRoot, absolute).split(path.sep).join('/');
        if (!isPage(absolute) && !relative.startsWith('..')) {
          node.url = `${blobUrl}/${relative}${hash ? `#${hash}` : ''}`;
        }
      }
      node.children?.forEach(visit);
    };
    visit(tree);
  };
}
