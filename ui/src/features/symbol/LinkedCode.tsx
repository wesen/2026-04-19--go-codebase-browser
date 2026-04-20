// React namespace provided by jsx: react-jsx
import { Link } from 'react-router-dom';
import { Code } from '../../packages/ui/src/Code';
import type { CodeRef } from '../../packages/ui/src/Code';

export interface LinkedCodeProps {
  text: string;
  refs?: CodeRef[];
  language?: string;
}

/**
 * LinkedCode wraps the widget-package <Code> with a React-Router-aware link
 * renderer. Kept in the app layer so the widget package stays router-free.
 */
export function LinkedCode({ text, refs, language = 'go' }: LinkedCodeProps) {
  return (
    <Code
      text={text}
      language={language}
      refs={refs}
      renderRefLink={(symbolId, children) => (
        <Link to={`/symbol/${encodeURIComponent(symbolId)}`} data-role="xref">
          {children}
        </Link>
      )}
    />
  );
}
