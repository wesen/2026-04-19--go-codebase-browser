import React from 'react';
import { useGetSymbolHistoryQuery } from '../../../api/historyApi';
import { SymbolDiffInlineWidget } from './SymbolDiffInlineWidget';

interface SymbolHistoryInlineWidgetProps {
  sym: string;
  limit?: number;
}

export function SymbolHistoryInlineWidget({ sym, limit = 8 }: SymbolHistoryInlineWidgetProps) {
  const { data, isLoading, error } = useGetSymbolHistoryQuery({ symbolId: sym, limit });
  const [selectedIndex, setSelectedIndex] = React.useState<number | null>(null);

  if (isLoading) {
    return (
      <section data-part="doc-snippet">
        <pre data-part="code-block"><code>Loading symbol history…</code></pre>
      </section>
    );
  }
  if (error) {
    return (
      <section data-part="doc-snippet">
        <div data-part="error">Failed to load symbol history: {JSON.stringify(error)}</div>
      </section>
    );
  }
  const entries = data ?? [];
  if (entries.length === 0) {
    return (
      <section data-part="doc-snippet">
        <div data-part="empty">No history found for <code>{sym}</code>.</div>
      </section>
    );
  }

  const selected = selectedIndex !== null ? entries[selectedIndex] : null;
  const previous = selectedIndex !== null && selectedIndex < entries.length - 1 ? entries[selectedIndex + 1] : null;
  const changedCount = entries.filter((entry, i) => {
    const prev = i < entries.length - 1 ? entries[i + 1] : undefined;
    return !!entry.bodyHash && entry.bodyHash !== prev?.bodyHash;
  }).length;

  return (
    <section data-part="doc-snippet" data-role="history">
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
        <strong>History: <code>{entries[0].kind} {sym.split('.').pop()}</code></strong>
        <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
          {entries.length} commit(s), {changedCount} body change(s)
        </span>
      </div>
      <div style={{ border: '1px solid var(--cb-color-border)', borderRadius: 8, overflow: 'hidden' }}>
        {entries.map((entry, i) => {
          const prev = i < entries.length - 1 ? entries[i + 1] : undefined;
          const changed = !!entry.bodyHash && entry.bodyHash !== prev?.bodyHash;
          const selectedRow = selectedIndex === i;
          const date = new Date(entry.authorTime * 1000).toISOString().slice(0, 10);
          return (
            <button
              key={entry.commitHash}
              type="button"
              onClick={() => setSelectedIndex(selectedRow ? null : i)}
              style={{
                display: 'grid',
                gridTemplateColumns: '1.5rem 5.5rem 5rem 1fr 4.5rem',
                gap: 8,
                width: '100%',
                textAlign: 'left',
                alignItems: 'baseline',
                border: 0,
                borderBottom: i === entries.length - 1 ? 0 : '1px solid var(--cb-color-border)',
                background: selectedRow ? 'rgba(33, 150, 243, 0.10)' : changed ? 'rgba(255, 152, 0, 0.08)' : 'transparent',
                color: 'var(--cb-color-text)',
                padding: '6px 8px',
                cursor: 'pointer',
                font: 'inherit',
              }}
              aria-expanded={selectedRow}
            >
              <span aria-label={changed ? 'body changed' : 'unchanged'}>{changed ? '●' : '○'}</span>
              <code>{entry.shortHash}</code>
              <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>{date}</span>
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{entry.message}</span>
              <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>{entry.startLine}-{entry.endLine}</span>
            </button>
          );
        })}
      </div>
      <div style={{ marginTop: 8, fontSize: 12, color: 'var(--cb-color-muted)' }}>
        ● = body changed compared with the previous indexed commit. Click a row to diff against its predecessor.
      </div>
      {selected && previous && selected.bodyHash !== previous.bodyHash && (
        <div style={{ marginTop: 12 }}>
          <SymbolDiffInlineWidget sym={sym} from={previous.commitHash} to={selected.commitHash} />
        </div>
      )}
      {selected && !previous && (
        <div style={{ marginTop: 12, color: 'var(--cb-color-muted)' }}>No predecessor commit available in this history window.</div>
      )}
    </section>
  );
}
