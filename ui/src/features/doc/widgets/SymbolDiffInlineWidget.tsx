import { useGetSymbolBodyDiffQuery } from '../../../api/historyApi';
import { DiffsUnifiedDiff } from '../../diff/DiffsUnifiedDiff';
import { HistoryUnavailableNotice, isHistoryUnavailable } from './historyUnavailable';

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
    if (isHistoryUnavailable(error)) {
      return <HistoryUnavailableNotice widget="Semantic diff" />;
    }
    return (
      <section data-part="doc-snippet">
        <div data-part="error">Failed to load diff: {JSON.stringify(error)}</div>
      </section>
    );
  }
  if (!data) return null;

  return (
    <section data-part="doc-snippet" data-role="diff">
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
        <strong>Diff: <code>{data.name || sym}</code></strong>
        <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
          <code>{from.slice(0, 7)}</code> → <code>{to.slice(0, 7)}</code>
          {' '}({data.oldRange} → {data.newRange})
        </span>
      </div>
      <DiffsUnifiedDiff
        name={`${data.name || 'symbol'}.go`}
        oldText={data.oldBody}
        newText={data.newBody}
        language="go"
        oldLabel={from.slice(0, 7)}
        newLabel={to.slice(0, 7)}
      />
    </section>
  );
}
