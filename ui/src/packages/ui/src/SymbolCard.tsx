import React from 'react';
import { PARTS } from './parts';
import type { Symbol } from '../../../api/types';

export interface SymbolCardProps {
  symbol: Symbol;
  snippet?: string;
  /** Render prop for linking the name to a navigation target. */
  renderName?: (name: string, id: string) => React.ReactNode;
}

export function SymbolCard({ symbol, snippet, renderName }: SymbolCardProps) {
  return (
    <article
      data-part={PARTS.symbolCard}
      data-state={snippet ? 'with-snippet' : 'no-snippet'}
    >
      <header data-part={PARTS.symbolHeader}>
        <span data-part={PARTS.symbolKind} data-role={symbol.kind}>
          {symbol.kind}
        </span>
        <code data-part={PARTS.symbolName}>
          {renderName ? renderName(symbol.name, symbol.id) : symbol.name}
        </code>
        {symbol.signature && (
          <code data-part={PARTS.symbolSignature}>
            {truncate(symbol.signature, 160)}
          </code>
        )}
      </header>
      {symbol.doc && (
        <div data-part={PARTS.symbolDoc} data-role="doc">
          {symbol.doc}
        </div>
      )}
      {snippet && (
        <pre data-part={PARTS.symbolSnippet} data-role="code">
          <code>{snippet}</code>
        </pre>
      )}
    </article>
  );
}

function truncate(s: string, n: number) {
  return s.length > n ? s.slice(0, n - 1) + '…' : s;
}
