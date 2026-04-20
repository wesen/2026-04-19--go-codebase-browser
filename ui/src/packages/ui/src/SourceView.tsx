// React namespace provided by jsx: react-jsx
import { PARTS } from './parts';
import { tokensByLine } from './highlight/go';
import { annotateComment } from './highlight/annotations';

export interface SourceViewProps {
  source: string;
  /** Optional 1-based line number to highlight. */
  highlightLine?: number;
  /** Language hint; 'go' (default) enables syntax highlighting. */
  language?: string;
}

export function SourceView({ source, highlightLine, language = 'go' }: SourceViewProps) {
  const lines = language === 'go' ? tokensByLine(source) : source.split('\n').map((l) => [{ type: 'id' as const, text: l }]);
  return (
    <div data-part={PARTS.sourceView} data-role={language}>
      {lines.map((tokens, i) => {
        const n = i + 1;
        return (
          <div
            key={n}
            data-part={PARTS.sourceLine}
            data-state={n === highlightLine ? 'highlight' : undefined}
            id={`L${n}`}
          >
            <span data-part={PARTS.sourceGutter}>{n.toString().padStart(4, ' ')}</span>
            <span data-role="content">
              {tokens.length === 0 ? '\u00a0' : tokens.map((t, k) => {
                if (t.type !== 'com') {
                  return <span key={k} data-tok={t.type}>{t.text}</span>;
                }
                const spans = annotateComment(t.text);
                return (
                  <span key={k} data-tok="com">
                    {spans.map((s, m) => (
                      <span key={m} data-annotation={s.annotation}>{s.text}</span>
                    ))}
                  </span>
                );
              })}
            </span>
          </div>
        );
      })}
    </div>
  );
}
