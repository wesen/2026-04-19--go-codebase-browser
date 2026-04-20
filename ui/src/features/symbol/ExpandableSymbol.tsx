// React namespace provided by jsx: react-jsx
import { useState } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { SymbolCard } from '../../packages/ui/src/SymbolCard';
import { useGetSnippetQuery } from '../../api/sourceApi';
import type { Symbol } from '../../api/types';

export interface ExpandableSymbolProps {
  symbol: Symbol;
  /** Kind of snippet to load when expanded; defaults to 'declaration'. */
  snippetKind?: 'declaration' | 'body' | 'signature';
  /** Pre-expanded by default (useful for stories). */
  defaultOpen?: boolean;
}

/**
 * ExpandableSymbol wraps SymbolCard with a lazy snippet loader. The snippet
 * query is skipped until the user opens the card, so rendering a long list
 * of symbols is cheap. Clicking "open" fetches /api/snippet once; subsequent
 * toggles reuse RTK-Query's cache.
 */
export function ExpandableSymbol({ symbol, snippetKind = 'declaration', defaultOpen = false }: ExpandableSymbolProps) {
  const [open, setOpen] = useState(defaultOpen);
  const { data: snippet, isFetching } = useGetSnippetQuery(
    { sym: symbol.id, kind: snippetKind },
    { skip: !open },
  );

  const actions: ReactNode = (
    <>
      <button
        type="button"
        data-part="symbol-toggle"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        {open ? 'hide code' : 'show code'}
      </button>{' '}
      <Link
        to={`/symbol/${encodeURIComponent(symbol.id)}`}
        data-part="symbol-toggle"
      >
        open →
      </Link>
    </>
  );

  const effectiveSnippet = open ? (isFetching ? 'loading…' : snippet) : undefined;

  return (
    <SymbolCard
      symbol={symbol}
      snippet={effectiveSnippet}
      renderName={(name, id) => <Link to={`/symbol/${encodeURIComponent(id)}`}>{name}</Link>}
      actions={actions}
    />
  );
}
