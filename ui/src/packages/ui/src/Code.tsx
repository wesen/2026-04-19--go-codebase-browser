// React namespace provided by jsx: react-jsx
import { tokenize, type Token } from './highlight/go';
import { annotateComment } from './highlight/annotations';

export interface CodeProps {
  text: string;
  /** Language hint; only 'go' is highlighted today. */
  language?: string;
}

/**
 * Code renders an inline-highlighted snippet inside a <pre><code> pair.
 * Use for short snippets (symbol cards, doc-page fences). For whole files
 * prefer SourceView which adds gutter + line-level structure.
 *
 * Comment tokens (// and /* *​/) are post-processed for godoc annotations
 * (Deprecated/TODO/BUG/FIXME/NOTE) so authors get a visible highlight
 * without having to scan the comment prose manually.
 */
export function Code({ text, language = 'go' }: CodeProps) {
  const tokens = language === 'go' ? tokenize(text) : [{ type: 'id', text } as Token];
  return (
    <pre data-part="code-block" data-role={language}>
      <code>{tokens.map(renderToken)}</code>
    </pre>
  );
}

function renderToken(t: Token, i: number) {
  if (t.type !== 'com') {
    return <span key={i} data-tok={t.type}>{t.text}</span>;
  }
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
