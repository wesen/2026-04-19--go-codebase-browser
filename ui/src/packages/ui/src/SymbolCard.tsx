// React namespace provided by jsx: react-jsx
import type { ReactNode } from 'react';
import { PARTS } from './parts';
import { Code } from './Code';
import { detectLeadingAnnotation } from './highlight/annotations';
import type { Symbol } from '../../../api/types';

export interface SymbolCardProps {
  symbol: Symbol;
  snippet?: string;
  /** Render prop for linking the name to a navigation target. */
  renderName?: (name: string, id: string) => ReactNode;
  /** Optional trailing slot for custom action buttons (e.g. expand/collapse). */
  actions?: ReactNode;
}

export function SymbolCard({ symbol, snippet, renderName, actions }: SymbolCardProps) {
  const annotation = symbol.doc ? detectLeadingAnnotation(symbol.doc) : undefined;
  return (
    <article
      data-part={PARTS.symbolCard}
      data-state={snippet ? 'with-snippet' : 'no-snippet'}
      data-annotation={annotation}
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
        {annotation === 'deprecated' && (
          <span data-part={PARTS.deprecatedBadge} data-role="deprecated">deprecated</span>
        )}
        {actions && <span data-role="actions" style={{ marginLeft: 'auto' }}>{actions}</span>}
      </header>
      {symbol.doc && (
        <div data-part={PARTS.symbolDoc} data-role="doc">
          {symbol.doc}
        </div>
      )}
      {snippet && <Code text={snippet} />}
    </article>
  );
}

function truncate(s: string, n: number) {
  return s.length > n ? s.slice(0, n - 1) + '…' : s;
}
