// React namespace provided by jsx: react-jsx
import { PARTS } from './parts';
import { Code } from './Code';

export interface SnippetProps {
  text: string;
  language?: string;
  /** When non-empty, rendered as a "jump to source" link. */
  jumpTo?: string;
}

export function Snippet({ text, language = 'go', jumpTo }: SnippetProps) {
  return (
    <div data-part={PARTS.snippetEmbed} data-role={language}>
      {jumpTo && (
        <a href={jumpTo} data-role="jump">
          ↪ jump to source
        </a>
      )}
      <Code text={text} language={language} />
    </div>
  );
}
