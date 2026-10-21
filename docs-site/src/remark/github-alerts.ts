// Plugin remark : rend les alertes GitHub (`> [!NOTE]`, `> [!WARNING]`…) en encadrés Docusaurus.
//
// Les documents lus sur place (frontend/docs/, frontend/README.md) se lisent aussi sur GitHub, qui ne
// connaît pas la syntaxe `:::note` de Docusaurus. On écrit donc la syntaxe de GitHub, et ce plugin la
// traduit en directive `:::` avant que Docusaurus ne transforme les directives en encadrés : à placer
// dans `beforeDefaultRemarkPlugins`.

type Node = {
  type: string;
  value?: string;
  children?: Node[];
  [key: string]: unknown;
};

// Type d'alerte GitHub → type d'encadré Docusaurus.
const ADMONITIONS: Record<string, string> = {
  NOTE: 'note',
  TIP: 'tip',
  IMPORTANT: 'info',
  WARNING: 'warning',
  CAUTION: 'danger',
};

const MARKER = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\][ \t]*(\r?\n)?/;

function toAdmonition(node: Node): Node | null {
  const paragraph = node.children?.[0];
  const text = paragraph?.type === 'paragraph' ? paragraph.children?.[0] : undefined;
  const match = text?.type === 'text' ? MARKER.exec(text.value ?? '') : null;
  if (!paragraph || !text || !match) {
    return null;
  }

  text.value = (text.value ?? '').slice(match[0].length);
  if (text.value === '') {
    paragraph.children!.shift();
  }
  const children = paragraph.children!.length > 0 ? node.children! : node.children!.slice(1);

  return {
    type: 'containerDirective',
    name: ADMONITIONS[match[1]],
    attributes: {},
    children,
    position: node.position,
  };
}

function transform(node: Node): void {
  node.children?.forEach((child, index) => {
    if (child.type === 'blockquote') {
      const admonition = toAdmonition(child);
      if (admonition) {
        node.children![index] = admonition;
        child = admonition;
      }
    }
    transform(child);
  });
}

export default function githubAlerts() {
  return (tree: Node) => transform(tree);
}
