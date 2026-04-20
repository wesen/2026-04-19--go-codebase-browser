// React namespace provided by jsx: react-jsx
import { Link } from 'react-router-dom';
import { useGetXrefQuery } from '../../api/xrefApi';

export interface XrefPanelProps {
  symbolId: string;
}

/**
 * XrefPanel shows two columns: who uses this symbol (usedBy) and what this
 * symbol references inside its body (uses). Both resolve to links.
 */
export function XrefPanel({ symbolId }: XrefPanelProps) {
  const { data, isLoading, error } = useGetXrefQuery(symbolId);
  if (isLoading) return <div data-part="loading">Loading xrefs…</div>;
  if (error) return <div data-part="error">Failed to load xrefs</div>;
  if (!data) return null;

  return (
    <section style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginTop: 16 }}>
      <div>
        <h3 style={{ marginTop: 0 }}>Used by ({data.usedBy.length})</h3>
        {data.usedBy.length === 0 ? (
          <div data-part="empty">No callers in index.</div>
        ) : (
          <ul data-part="tree-nav">
            {data.usedBy.map((r, i) => (
              <li key={i}>
                <Link data-part="tree-node" to={`/symbol/${encodeURIComponent(r.fromSymbolId)}`}>
                  <span data-part="symbol-kind" data-role={r.kind}>{r.kind}</span>{' '}
                  <code>{shortenID(r.fromSymbolId)}</code>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
      <div>
        <h3 style={{ marginTop: 0 }}>Uses ({data.uses.length})</h3>
        {data.uses.length === 0 ? (
          <div data-part="empty">This symbol is a leaf (no outgoing refs).</div>
        ) : (
          <ul data-part="tree-nav">
            {data.uses.map((u, i) => (
              <li key={i}>
                <Link data-part="tree-node" to={`/symbol/${encodeURIComponent(u.toSymbolId)}`}>
                  <span data-part="symbol-kind" data-role={u.kind}>{u.kind}</span>{' '}
                  <code>{shortenID(u.toSymbolId)}</code>{' '}
                  {u.count > 1 && <span data-role="hint">×{u.count}</span>}
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}

// shortenID extracts the last ~2 path segments for display.
// sym:github.com/wesen/codebase-browser/internal/indexer.func.SymbolID ->
//   internal/indexer.func.SymbolID
function shortenID(id: string): string {
  const body = id.replace(/^sym:/, '');
  const parts = body.split('/');
  return parts.slice(-2).join('/');
}
