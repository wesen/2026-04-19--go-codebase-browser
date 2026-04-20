// React namespace provided by jsx: react-jsx
import { tokenize, type Token } from './highlight/go';

export interface CodeProps {
  text: string;
  /** Language hint; only 'go' is highlighted today. */
  language?: string;
}

/**
 * Code renders an inline-highlighted snippet inside a <pre><code> pair.
 * Use for short snippets (symbol cards, doc-page fences). For whole files
 * prefer SourceView which adds gutter + line-level structure.
 */
export function Code({ text, language = 'go' }: CodeProps) {
  const tokens = language === 'go' ? tokenize(text) : [{ type: 'id', text } as Token];
  return (
    <pre data-part="code-block" data-role={language}>
      <code>
        {tokens.map((t, i) => (
          <span key={i} data-tok={t.type}>
            {t.text}
          </span>
        ))}
      </code>
    </pre>
  );
}
