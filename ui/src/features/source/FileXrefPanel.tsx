// React namespace provided by jsx: react-jsx
import { Link } from 'react-router-dom';
import { useGetFileXrefQuery } from '../../api/sourceApi';

export interface FileXrefPanelProps {
  path: string;
}

/**
 * FileXrefPanel shows "used by" / "uses" aggregated across every symbol
 * declared in a file. Mirrors XrefPanel's layout but operates on a file
 * rather than a single symbol; intra-file refs are dropped server-side.
 */
export function FileXrefPanel({ path }: FileXrefPanelProps) {
  const { data, isLoading, error } = useGetFileXrefQuery(path, { skip: !path });
  if (isLoading) return <div data-part="loading">Loading xrefs…</div>;
  if (error) return <div data-part="error">Failed to load xrefs</div>;
  if (!data) return null;

  return (
    <section style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginTop: 24 }}>
      <div>
        <h3 style={{ marginTop: 0 }}>Used by ({data.usedBy.length})</h3>
        {data.usedBy.length === 0 ? (
          <div data-part="empty">Nothing outside this file references its symbols.</div>
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
          <div data-part="empty">This file's symbols don't reference anything else in the index.</div>
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

function shortenID(id: string): string {
  const body = id.replace(/^sym:/, '');
  const parts = body.split('/');
  return parts.slice(-2).join('/');
}
