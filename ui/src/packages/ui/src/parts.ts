export const WIDGET_ROOT = 'codebase-browser';

export const PARTS = {
  root: 'root',
  layout: 'layout',
  sidebar: 'sidebar',
  main: 'main',
  treeNav: 'tree-nav',
  treeNode: 'tree-node',
  treeNodeLeaf: 'tree-node-leaf',
  treeNodeExpanded: 'tree-node-expanded',
  symbolCard: 'symbol-card',
  symbolHeader: 'symbol-header',
  symbolKind: 'symbol-kind',
  symbolName: 'symbol-name',
  symbolSignature: 'symbol-signature',
  symbolDoc: 'symbol-doc',
  symbolSnippet: 'symbol-snippet',
  sourceView: 'source-view',
  sourceLine: 'source-line',
  sourceGutter: 'source-gutter',
  codeBlock: 'code-block',
  symbolToggle: 'symbol-toggle',
  docPage: 'doc-page',
  snippetEmbed: 'snippet-embed',
  searchBox: 'search-box',
  searchResult: 'search-result',
  empty: 'empty',
  loading: 'loading',
  error: 'error',
} as const;

export type PartName = (typeof PARTS)[keyof typeof PARTS];

// Helper: the widget root attribute. Put this on the outermost element so
// all theme selectors `[data-widget="codebase-browser"]` can hook in.
export const widgetRootAttrs = { 'data-widget': WIDGET_ROOT } as const;
