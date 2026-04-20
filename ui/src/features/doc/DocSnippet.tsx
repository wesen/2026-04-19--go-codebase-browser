// React namespace provided by jsx: react-jsx
import { Link } from 'react-router-dom';
import { useGetSymbolQuery } from '../../api/indexApi';
import { useGetSnippetQuery, useGetSnippetRefsQuery } from '../../api/sourceApi';
import { LinkedCode } from '../symbol/LinkedCode';

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

export function DocSnippet({ sym, directive, lang }: DocSnippetProps) {
  if (directive === 'codebase-signature') return <DocSignature sym={sym} />;
  if (directive === 'codebase-doc') return <DocGodoc sym={sym} />;
  return <DocFullSnippet sym={sym} lang={lang} />;
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

function DocFullSnippet({ sym, lang }: { sym: string; lang: string }) {
  const { data: symbol } = useGetSymbolQuery(sym);
  const { data: snippet } = useGetSnippetQuery({ sym });
  const { data: refs } = useGetSnippetRefsQuery(sym);
  if (!symbol || snippet === undefined) {
    return (
      <pre data-part="code-block">
        <code>Loading…</code>
      </pre>
    );
  }
  const language = lang || symbol.language || 'go';
  return (
    <section data-part="doc-snippet">
      <header data-part="symbol-header">
        <span data-part="symbol-kind" data-role={symbol.kind}>
          {symbol.kind}
        </span>
        <Link
          to={`/symbol/${encodeURIComponent(sym)}`}
          data-part="symbol-name"
          data-role="xref"
        >
          <code>{symbol.name}</code>
        </Link>
        {symbol.signature && (
          <code data-part="symbol-signature" data-role="hint">
            {symbol.signature}
          </code>
        )}
      </header>
      <LinkedCode text={snippet} refs={refs} language={language} />
    </section>
  );
}
