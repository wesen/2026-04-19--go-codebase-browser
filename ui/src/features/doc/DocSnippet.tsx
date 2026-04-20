// React namespace provided by jsx: react-jsx
import { Link } from 'react-router-dom';
import { useGetSymbolQuery } from '../../api/indexApi';
import { ExpandableSymbol } from '../symbol/ExpandableSymbol';
import { XrefPanel } from '../symbol/XrefPanel';

/**
 * DocSnippet hydrates one `[data-codebase-snippet]` stub on a doc page.
 * The server-rendered stub carries the symbol id + directive type; we
 * dispatch to the right widget so every directive on a doc page gets
 * the same interactive treatment a /symbol/{id} page does:
 *
 *   - codebase-snippet    → <LinkedCode> with clickable xrefs
 *   - codebase-signature  → compact <Link> to the symbol
 *   - codebase-doc        → blockquote of the godoc
 */
export interface DocSnippetProps {
  sym: string;
  directive: string;
  kind: string;
  lang: string;
}

export function DocSnippet({ sym, directive }: DocSnippetProps) {
  if (directive === 'codebase-signature') return <DocSignature sym={sym} />;
  if (directive === 'codebase-doc') return <DocGodoc sym={sym} />;
  return <DocFullSnippet sym={sym} />;
}

function DocSignature({ sym }: { sym: string }) {
  const { data } = useGetSymbolQuery(sym);
  return (
    <pre data-part="code-block" data-role="signature">
      <Link to={`/symbol/${encodeURIComponent(sym)}`} data-role="xref">
        <code data-tok="kw">{data?.signature ?? data?.name ?? sym}</code>
      </Link>
    </pre>
  );
}

function DocGodoc({ sym }: { sym: string }) {
  const { data } = useGetSymbolQuery(sym);
  return (
    <blockquote data-part="symbol-doc" data-role="doc">
      {data?.doc ?? ''}
    </blockquote>
  );
}

// DocFullSnippet wraps <ExpandableSymbol> (same component used on symbol
// pages) and adds a collapsible cross-reference panel beneath it. Readers
// get the same rich, navigable view of an embedded snippet that they'd see
// on /symbol/{id} — plus a show/hide toggle so long snippets can collapse
// once they've been skimmed, and a <details> section for digging into
// callers and callees without leaving the doc page.
function DocFullSnippet({ sym }: { sym: string }) {
  const { data: symbol } = useGetSymbolQuery(sym);
  if (!symbol) {
    return (
      <pre data-part="code-block">
        <code>Loading…</code>
      </pre>
    );
  }
  return (
    <section data-part="doc-snippet">
      <ExpandableSymbol symbol={symbol} defaultOpen />
      <details data-part="doc-snippet-xref" style={{ marginTop: 8 }}>
        <summary data-role="hint" style={{ cursor: 'pointer' }}>
          cross-references
        </summary>
        <XrefPanel symbolId={sym} />
      </details>
    </section>
  );
}
