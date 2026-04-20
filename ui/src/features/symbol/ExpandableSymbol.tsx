// React namespace provided by jsx: react-jsx
import { useState } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { PARTS } from '../../packages/ui/src/parts';
import { detectLeadingAnnotation } from '../../packages/ui/src/highlight/annotations';
import { LinkedCode } from './LinkedCode';
import { useGetSnippetQuery, useGetSnippetRefsQuery } from '../../api/sourceApi';
import type { Symbol } from '../../api/types';

export interface ExpandableSymbolProps {
  symbol: Symbol;
  snippetKind?: 'declaration' | 'body' | 'signature';
  defaultOpen?: boolean;
}

/**
 * ExpandableSymbol renders a SymbolCard-like surface with a show/hide toggle
 * and, when open, fetches both the snippet text and its linkified refs in
 * parallel. Identifier tokens that match indexed symbols become in-snippet
 * navigation links.
 *
 * We inline the card markup here (instead of using the presentational
 * SymbolCard) because we need to replace the snippet <pre> with LinkedCode
 * and SymbolCard's snippet slot takes a plain string.
 */
export function ExpandableSymbol({ symbol, snippetKind = 'declaration', defaultOpen = false }: ExpandableSymbolProps) {
  const [open, setOpen] = useState(defaultOpen);
  const { data: snippet, isFetching: snipLoading } = useGetSnippetQuery(
    { sym: symbol.id, kind: snippetKind },
    { skip: !open },
  );
  const { data: refs } = useGetSnippetRefsQuery(symbol.id, { skip: !open });
  const annotation = symbol.doc ? detectLeadingAnnotation(symbol.doc) : undefined;

  const actions: ReactNode = (
    <span data-role="actions" style={{ marginLeft: 'auto' }}>
      <button
        type="button"
        data-part={PARTS.symbolToggle}
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        {open ? 'hide code' : 'show code'}
      </button>{' '}
      <Link to={`/symbol/${encodeURIComponent(symbol.id)}`} data-part={PARTS.symbolToggle}>
        open →
      </Link>
    </span>
  );

  return (
    <article
      data-part={PARTS.symbolCard}
      data-state={open ? 'with-snippet' : 'no-snippet'}
      data-annotation={annotation}
    >
      <header data-part={PARTS.symbolHeader}>
        <span data-part={PARTS.symbolKind} data-role={symbol.kind}>
          {symbol.kind}
        </span>
        <code data-part={PARTS.symbolName}>
          <Link to={`/symbol/${encodeURIComponent(symbol.id)}`}>{symbol.name}</Link>
        </code>
        {symbol.signature && (
          <code data-part={PARTS.symbolSignature}>
            {truncate(symbol.signature, 160)}
          </code>
        )}
        {annotation === 'deprecated' && (
          <span data-part={PARTS.deprecatedBadge} data-role="deprecated">deprecated</span>
        )}
        {actions}
      </header>
      {symbol.doc && (
        <div data-part={PARTS.symbolDoc} data-role="doc">
          {symbol.doc}
        </div>
      )}
      {open && (snipLoading ? (
        <div data-part="loading">Loading snippet…</div>
      ) : snippet ? (
        <LinkedCode text={snippet} refs={refs} language={symbol.language ?? 'go'} />
      ) : null)}
    </article>
  );
}

function truncate(s: string, n: number) {
  return s.length > n ? s.slice(0, n - 1) + '…' : s;
}
