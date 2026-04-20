// React namespace provided by jsx: react-jsx
import { PARTS } from './parts';

export interface SnippetProps {
  text: string;
  language?: string;
  /** When non-empty, rendered as a "jump to source" link. */
  jumpTo?: string;
}

export function Snippet({ text, language, jumpTo }: SnippetProps) {
  return (
    <div data-part={PARTS.snippetEmbed} data-role={language ?? 'go'}>
      {jumpTo && (
        <a href={jumpTo} data-role="jump">
          ↪ jump to source
        </a>
      )}
      <pre>
        <code>{text}</code>
      </pre>
    </div>
  );
}
