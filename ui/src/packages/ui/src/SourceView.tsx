// React namespace provided by jsx: react-jsx
import { PARTS } from './parts';

export interface SourceViewProps {
  source: string;
  /** Optional 1-based line number to highlight. */
  highlightLine?: number;
}

export function SourceView({ source, highlightLine }: SourceViewProps) {
  const lines = source.split('\n');
  return (
    <div data-part={PARTS.sourceView}>
      {lines.map((line, i) => {
        const n = i + 1;
        return (
          <div
            key={n}
            data-part={PARTS.sourceLine}
            data-state={n === highlightLine ? 'highlight' : undefined}
            id={`L${n}`}
          >
            <span data-part={PARTS.sourceGutter}>{n.toString().padStart(4, ' ')}</span>
            <span data-role="content">{line || '\u00a0'}</span>
          </div>
        );
      })}
    </div>
  );
}
