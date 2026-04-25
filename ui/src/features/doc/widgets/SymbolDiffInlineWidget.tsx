import { useGetSymbolBodyDiffQuery } from '../../../api/historyApi';

interface SymbolDiffInlineWidgetProps {
  sym: string;
  from: string;
  to: string;
}

export function SymbolDiffInlineWidget({ sym, from, to }: SymbolDiffInlineWidgetProps) {
  const { data, isLoading, error } = useGetSymbolBodyDiffQuery(
    { from, to, symbolId: sym },
    { skip: !sym || !from || !to },
  );

  if (isLoading) {
    return (
      <section data-part="doc-snippet">
        <pre data-part="code-block"><code>Loading diff…</code></pre>
      </section>
    );
  }
  if (error) {
    return (
      <section data-part="doc-snippet">
        <div data-part="error">Failed to load diff: {JSON.stringify(error)}</div>
      </section>
    );
  }
  if (!data) return null;

  const lines = (data.unifiedDiff || '').split('\n');

  return (
    <section data-part="doc-snippet" data-role="diff">
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
        <strong>Diff: <code>{data.name || sym}</code></strong>
        <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
          <code>{from.slice(0, 7)}</code> → <code>{to.slice(0, 7)}</code>
          {' '}({data.oldRange} → {data.newRange})
        </span>
      </div>
      <pre
        data-part="code-block"
        data-role="diff"
        style={{ whiteSpace: 'pre-wrap', maxHeight: '60vh', overflow: 'auto' }}
      >
        <code>
          {lines.map((line, i) => {
            const style = line.startsWith('- ')
              ? { background: 'rgba(244, 67, 54, 0.12)', color: '#c62828', display: 'block' }
              : line.startsWith('+ ')
                ? { background: 'rgba(76, 175, 80, 0.12)', color: '#2e7d32', display: 'block' }
                : { color: 'var(--cb-color-muted)', display: 'block' };
            return <span key={i} style={style}>{line || ' '}</span>;
          })}
        </code>
      </pre>
    </section>
  );
}
