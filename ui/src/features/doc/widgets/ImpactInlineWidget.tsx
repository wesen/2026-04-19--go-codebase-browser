import { Link } from 'react-router-dom';
import { useGetImpactQuery } from '../../../api/historyApi';

interface ImpactInlineWidgetProps {
  sym: string;
  dir?: 'usedby' | 'uses';
  depth?: number;
  commit?: string;
}

export function ImpactInlineWidget({ sym, dir = 'usedby', depth = 2, commit }: ImpactInlineWidgetProps) {
  const { data, isLoading, error } = useGetImpactQuery({ sym, dir, depth, commit });

  if (isLoading) {
    return (
      <section data-part="doc-snippet">
        <pre data-part="code-block"><code>Loading impact…</code></pre>
      </section>
    );
  }
  if (error) {
    return (
      <section data-part="doc-snippet">
        <div data-part="error">Failed to load impact: {JSON.stringify(error)}</div>
      </section>
    );
  }
  if (!data) return null;

  const grouped = new Map<number, typeof data.nodes>();
  for (const node of data.nodes) {
    const bucket = grouped.get(node.depth) ?? [];
    bucket.push(node);
    grouped.set(node.depth, bucket);
  }

  return (
    <section data-part="doc-snippet" data-role="impact">
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
        <strong>Impact: <code>{sym.split('.').pop()}</code></strong>
        <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
          {dir === 'usedby' ? 'used by' : 'uses'} · depth {data.depth} · {data.nodes.length} symbol(s)
        </span>
      </div>
      {data.nodes.length === 0 ? (
        <div data-part="empty" style={{ border: '1px dashed var(--cb-color-border)', borderRadius: 8, padding: 12 }}>
          No {dir === 'usedby' ? 'callers' : 'callees'} found at this commit.
        </div>
      ) : (
        <div style={{ display: 'grid', gap: 12 }}>
          {[...grouped.entries()].sort((a, b) => a[0] - b[0]).map(([d, nodes]) => (
            <div key={d} style={{ border: '1px solid var(--cb-color-border)', borderRadius: 8, overflow: 'hidden' }}>
              <div style={{ padding: '6px 8px', fontWeight: 700, background: 'rgba(127, 127, 127, 0.08)' }}>
                Depth {d} — {nodes.length} symbol(s)
              </div>
              {nodes.map((node) => (
                <div
                  key={node.symbolId}
                  style={{
                    display: 'grid',
                    gridTemplateColumns: '5rem 1fr 6rem 4rem',
                    gap: 8,
                    alignItems: 'baseline',
                    borderTop: '1px solid var(--cb-color-border)',
                    padding: '6px 8px',
                  }}
                >
                  <code style={{ fontSize: 12 }}>{node.kind}</code>
                  <Link to={`/symbol/${encodeURIComponent(node.symbolId)}`} style={{ textDecoration: 'none' }}>
                    <code>{node.name}</code>
                  </Link>
                  <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
                    {node.edges.length} edge{node.edges.length === 1 ? '' : 's'}
                  </span>
                  <span style={{ fontSize: 12 }} title={`compatibility: ${node.compatibility}`}>
                    {node.compatibility === 'ok' ? '✓' : node.compatibility === 'review' ? '⚠' : '·'}
                  </span>
                </div>
              ))}
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
