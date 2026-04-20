import { tokenize as tokenizeGo, tokensByLine as tokensByLineGo } from './go';
import { tokenize as tokenizeTS, tokensByLine as tokensByLineTS } from './ts';
import type { Token } from './go';

export type { Token, TokenType } from './go';

/**
 * tokenizeForLanguage dispatches on a language tag. Unknown languages fall
 * back to a single `id` token so <Code> / <SourceView> still render the
 * raw text without highlighting.
 */
export function tokenizeForLanguage(lang: string | undefined, src: string): Token[] {
  switch (lang) {
    case 'go':
      return tokenizeGo(src);
    case 'ts':
    case 'tsx':
    case 'typescript':
      return tokenizeTS(src);
    default:
      return [{ type: 'id', text: src }];
  }
}

export function tokensByLineForLanguage(lang: string | undefined, src: string): Token[][] {
  switch (lang) {
    case 'go':
      return tokensByLineGo(src);
    case 'ts':
    case 'tsx':
    case 'typescript':
      return tokensByLineTS(src);
    default:
      return src.split('\n').map((line) => [{ type: 'id', text: line } as Token]);
  }
}
