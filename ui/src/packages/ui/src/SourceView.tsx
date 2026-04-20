// React namespace provided by jsx: react-jsx
import type { ReactNode } from 'react';
import { PARTS } from './parts';
import { tokensByLineForLanguage } from './highlight';
import { annotateComment } from './highlight/annotations';

export interface SourceRef {
  /** Byte offset into the source (not a snippet). */
  offset: number;
  length: number;
  toSymbolId: string;
}

export interface SourceViewProps {
  source: string;
  /** Optional 1-based line number to highlight. */
  highlightLine?: number;
  /** Language hint; 'go' (default) enables syntax highlighting. */
  language?: string;
  /** File-level identifier cross-references. When present, identifier tokens
   *  whose byte offset matches a ref are rendered as anchors via renderRefLink. */
  refs?: SourceRef[];
  /** Render prop for turning a ref target into a link — app layer supplies it. */
  renderRefLink?: (symbolId: string, children: ReactNode) => ReactNode;
}

export function SourceView({
  source,
  highlightLine,
  language = 'go',
  refs,
  renderRefLink,
}: SourceViewProps) {
  const lines = tokensByLineForLanguage(language, source);
  // Index refs by byte offset for O(1) lookup during the token walk.
  const refByOffset = new Map<number, SourceRef>();
  if (refs) for (const r of refs) refByOffset.set(r.offset, r);

  // tokensByLine strips the trailing newline from each line, so reconstruct
  // the file-level byte offset as `sum(token lengths) + one byte per line
  // break before the current line`.
  let byteOffset = 0;
  return (
    <div data-part={PARTS.sourceView} data-role={language}>
      {lines.map((tokens, i) => {
        const n = i + 1;
        if (i > 0) byteOffset += 1; // newline between previous and current line
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
                const tokenStart = byteOffset;
                byteOffset += t.text.length;
                if (t.type === 'com') {
                  const spans = annotateComment(t.text);
                  return (
                    <span key={k} data-tok="com">
                      {spans.map((s, m) => (
                        <span key={m} data-annotation={s.annotation}>{s.text}</span>
                      ))}
                    </span>
                  );
                }
                if (renderRefLink && (t.type === 'id' || t.type === 'type')) {
                  const ref = refByOffset.get(tokenStart);
                  if (ref) {
                    const inner = (
                      <span data-tok={t.type} data-role="ref">
                        {t.text}
                      </span>
                    );
                    return <span key={k}>{renderRefLink(ref.toSymbolId, inner)}</span>;
                  }
                }
                return <span key={k} data-tok={t.type}>{t.text}</span>;
              })}
            </span>
          </div>
        );
      })}
    </div>
  );
}
