import React from 'react';
import { Code } from '../../../packages/ui/src/Code';

interface AnnotationWidgetProps {
  sym: string;
  language?: string;
  commit?: string;
  lines?: string;
  note?: string;
}

export function AnnotationWidget({ sym, language = 'go', commit, lines, note }: AnnotationWidgetProps) {
  const [text, setText] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const controller = new AbortController();
    const params = new URLSearchParams({ sym, kind: 'declaration' });
    if (commit) params.set('commit', commit);
    fetch(`/api/snippet?${params.toString()}`, { signal: controller.signal })
      .then((resp) => {
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        return resp.text();
      })
      .then(setText)
      .catch((err) => {
        if (!controller.signal.aborted) setError(String(err));
      });
    return () => controller.abort();
  }, [sym, commit]);

  if (error) return <div data-part="error">Failed to load annotation snippet: {error}</div>;
  if (text === null) return <pre data-part="code-block"><code>Loading annotation…</code></pre>;

  const highlight = parseLineSpec(lines);
  const sourceLines = text.split('\n');
  return (
    <section data-part="doc-snippet" data-role="annotation">
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
        <strong>Annotated snippet: <code>{displayName(sym)}</code></strong>
        {commit && <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>at <code>{commit.slice(0, 7)}</code></span>}
      </div>
      <div style={{ display: 'grid', gap: 2 }}>
        {sourceLines.map((line, index) => {
          const lineNo = index + 1;
          const active = highlight.has(lineNo);
          return (
            <div
              key={lineNo}
              style={{
                display: 'grid',
                gridTemplateColumns: '3rem minmax(0, 1fr)',
                gap: 8,
                background: active ? 'rgba(255, 193, 7, 0.18)' : 'transparent',
                borderLeft: active ? '3px solid #ffb300' : '3px solid transparent',
                paddingLeft: 4,
              }}
            >
              <code style={{ userSelect: 'none', color: 'var(--cb-color-muted)', textAlign: 'right', paddingTop: 8 }}>{lineNo}</code>
              <Code text={line || ' '} language={language} />
            </div>
          );
        })}
      </div>
      {note && (
        <div style={{ marginTop: 8, border: '1px solid var(--cb-color-border)', borderRadius: 8, padding: 8 }}>
          <strong>Note:</strong> {note.split('_').join(' ')}
        </div>
      )}
    </section>
  );
}

function parseLineSpec(spec?: string): Set<number> {
  const result = new Set<number>();
  if (!spec) return result;
  for (const part of spec.split(',')) {
    const [startRaw, endRaw] = part.split('-');
    const start = Number.parseInt(startRaw, 10);
    const end = endRaw ? Number.parseInt(endRaw, 10) : start;
    if (!Number.isFinite(start) || !Number.isFinite(end)) continue;
    for (let n = Math.min(start, end); n <= Math.max(start, end); n++) result.add(n);
  }
  return result;
}

function displayName(symbolId: string): string {
  const trimmed = symbolId.startsWith('sym:') ? symbolId.slice(4) : symbolId;
  const dot = trimmed.lastIndexOf('.');
  return dot >= 0 ? trimmed.slice(dot + 1) : trimmed;
}
