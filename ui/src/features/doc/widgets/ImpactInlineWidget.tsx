import { Link } from 'react-router-dom';
import { useGetImpactQuery } from '../../../api/historyApi';
import { HistoryUnavailableNotice, isHistoryUnavailable } from './historyUnavailable';

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
    if (isHistoryUnavailable(error)) {
      return <HistoryUnavailableNotice widget="Impact analysis" />;
    }
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
  const localCount = data.nodes.filter((node) => node.local).length;
  const externalCount = data.nodes.length - localCount;

  return (
    <section data-part="doc-snippet" data-role="impact">
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
        <strong>Impact: <code>{displayName(sym)}</code></strong>
        <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
          {dir === 'usedby' ? 'used by' : 'uses'} · depth {data.depth} · {localCount} local
          {externalCount ? ` · ${externalCount} external` : ''}
          {' '}· <code>{data.commit.slice(0, 7)}</code>
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
                    gridTemplateColumns: '5.5rem minmax(0, 1fr) 6rem 5rem',
                    gap: 8,
                    alignItems: 'baseline',
                    borderTop: '1px solid var(--cb-color-border)',
                    padding: '6px 8px',
                  }}
                >
                  <code style={{ fontSize: 12, color: node.local ? 'inherit' : 'var(--cb-color-muted)' }}>
                    {node.kind}
                  </code>
                  <span style={{ minWidth: 0 }}>
                    {node.local ? (
                      <Link
                        to={`/history?symbol=${encodeURIComponent(node.symbolId)}`}
                        style={{ textDecoration: 'none' }}
                        title="Open symbol history (history-backed link)"
                      >
                        <code>{node.name}</code>
                      </Link>
                    ) : (
                      <code title={node.symbolId} style={{ color: 'var(--cb-color-muted)' }}>{node.name}</code>
                    )}
                    {!node.local && (
                      <span style={{ marginLeft: 6, fontSize: 11, color: 'var(--cb-color-muted)' }}>external</span>
                    )}
                  </span>
                  <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
                    {node.edges.length} edge{node.edges.length === 1 ? '' : 's'}
                  </span>
                  <span style={{ fontSize: 12 }} title={`compatibility: ${node.compatibility}`}>
                    {node.compatibility === 'ok' ? '✓ ok' : node.compatibility === 'review' ? '⚠ review' : '·'}
                  </span>
                </div>
              ))}
            </div>
          ))}
        </div>
      )}
      <div style={{ marginTop: 8, fontSize: 12, color: 'var(--cb-color-muted)' }}>
        Local names link to the history-backed symbol view so links still work when the static HEAD index is older than the history DB.
      </div>
    </section>
  );
}

function displayName(symbolId: string): string {
  const trimmed = symbolId.startsWith('sym:') ? symbolId.slice(4) : symbolId;
  const dot = trimmed.lastIndexOf('.');
  return dot >= 0 ? trimmed.slice(dot + 1) : trimmed;
}
