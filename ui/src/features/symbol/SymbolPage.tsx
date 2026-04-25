// React namespace provided by jsx: react-jsx
import { Link, useParams } from 'react-router-dom';
import { useGetSymbolQuery } from '../../api/indexApi';
import { useGetSnippetQuery, useGetSnippetRefsQuery } from '../../api/sourceApi';
import { PARTS } from '../../packages/ui/src/parts';
import { detectLeadingAnnotation } from '../../packages/ui/src/highlight/annotations';
import { LinkedCode } from './LinkedCode';
import { XrefPanel } from './XrefPanel';

export function SymbolPage() {
  const { id: rawId } = useParams<{ id: string }>();
  const id = rawId ? decodeURIComponent(rawId) : '';
  const { data: sym, isLoading, error } = useGetSymbolQuery(id, { skip: !id });
  const { data: snippet } = useGetSnippetQuery(
    { sym: id, kind: 'declaration' },
    { skip: !id },
  );
  const { data: refs } = useGetSnippetRefsQuery(id, { skip: !id });
  if (isLoading) return <div data-part="loading">Loading…</div>;
  if (error) return <div data-part="error">Failed to load symbol</div>;
  if (!sym) return <div data-part="empty">Symbol not found</div>;

  const annotation = sym.doc ? detectLeadingAnnotation(sym.doc) : undefined;

  return (
    <div>
      <div data-part="symbol-doc" style={{ fontSize: 12 }}>
        <Link to={`/packages/${encodeURIComponent(sym.packageId)}`}>
          {sym.packageId.replace(/^pkg:/, '')}
        </Link>
      </div>

      <article
        data-part={PARTS.symbolCard}
        data-state={snippet ? 'with-snippet' : 'no-snippet'}
        data-annotation={annotation}
      >
        <header data-part={PARTS.symbolHeader}>
          <span data-part={PARTS.symbolKind} data-role={sym.kind}>{sym.kind}</span>
          <code data-part={PARTS.symbolName}>{sym.name}</code>
          {sym.signature && (
            <code data-part={PARTS.symbolSignature}>{sym.signature}</code>
          )}
          {annotation === 'deprecated' && (
            <span data-part={PARTS.deprecatedBadge} data-role="deprecated">deprecated</span>
          )}
        </header>
        {sym.doc && (
          <div data-part={PARTS.symbolDoc} data-role="doc">{sym.doc}</div>
        )}
        {snippet && <LinkedCode text={snippet} refs={refs} language={sym.language ?? 'go'} />}
      </article>

      <p data-part="symbol-doc">
        File:{' '}
        <Link to={`/source/${sym.fileId.replace(/^file:/, '')}`}>
          {sym.fileId.replace(/^file:/, '')}
        </Link>{' '}
        (lines {sym.range.startLine}–{sym.range.endLine})
      </p>

      <XrefPanel symbolId={sym.id} />

      <div style={{ marginTop: 16 }}>
        <Link
          to={`/history?symbol=${encodeURIComponent(sym.id)}`}
          style={{ fontSize: 13, color: 'var(--cb-color-link, #2196f3)' }}
        >
          📜 View change history
        </Link>
      </div>
    </div>
  );
}
