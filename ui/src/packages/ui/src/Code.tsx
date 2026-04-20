// React namespace provided by jsx: react-jsx
import type { ReactNode } from 'react';
import { tokenize, type Token } from './highlight/go';
import { annotateComment } from './highlight/annotations';

export interface CodeRef {
  /** Byte offset into the rendered text. */
  offsetInSnippet: number;
  /** Length in bytes of the identifier use. */
  length: number;
  /** Target symbol ID. */
  toSymbolId: string;
}

export interface CodeProps {
  text: string;
  /** Language hint; only 'go' is highlighted today. */
  language?: string;
  /**
   * Optional identifier cross-references. When present, identifier tokens whose
   * offset exactly matches a ref are rendered as anchors. Using exact offset
   * match (rather than name lookup) avoids false positives on same-named
   * locals — the indexer already did the resolution work.
   */
  refs?: CodeRef[];
  /** Render prop for turning a ref target into an <a> — app layer supplies it. */
  renderRefLink?: (symbolId: string, children: ReactNode) => ReactNode;
}

/**
 * Code renders an inline-highlighted snippet inside a <pre><code> pair.
 * Comments get godoc-annotation spans; identifier uses get linkified when
 * a matching ref is provided.
 */
export function Code({ text, language = 'go', refs, renderRefLink }: CodeProps) {
  const tokens = language === 'go' ? tokenize(text) : [{ type: 'id', text } as Token];
  // Index refs by byte offset for O(1) lookup while walking tokens.
  const refByOffset = new Map<number, CodeRef>();
  if (refs) for (const r of refs) refByOffset.set(r.offsetInSnippet, r);

  let offset = 0;
  const children: ReactNode[] = [];
  tokens.forEach((t, i) => {
    children.push(renderToken(t, i, offset, refByOffset, renderRefLink));
    offset += t.text.length;
  });

  return (
    <pre data-part="code-block" data-role={language}>
      <code>{children}</code>
    </pre>
  );
}

function renderToken(
  t: Token,
  i: number,
  offset: number,
  refByOffset: Map<number, CodeRef>,
  renderRefLink?: (symbolId: string, children: ReactNode) => ReactNode,
): ReactNode {
  // Comment tokens may contain godoc annotations.
  if (t.type === 'com') {
    const spans = annotateComment(t.text);
    if (spans.length === 1 && !spans[0].annotation) {
      return <span key={i} data-tok="com">{t.text}</span>;
    }
    return (
      <span key={i} data-tok="com">
        {spans.map((s, k) => (
          <span key={k} data-annotation={s.annotation}>{s.text}</span>
        ))}
      </span>
    );
  }

  // Identifier tokens at a known ref offset become links.
  if (t.type === 'id' && renderRefLink) {
    const ref = refByOffset.get(offset);
    if (ref) {
      const inner = <span data-tok="id" data-role="ref">{t.text}</span>;
      return <span key={i}>{renderRefLink(ref.toSymbolId, inner)}</span>;
    }
  }

  return <span key={i} data-tok={t.type}>{t.text}</span>;
}
