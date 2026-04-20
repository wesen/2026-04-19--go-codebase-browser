// Minimal Go source tokenizer. Deliberately small and in-tree so the widget
// package stays zero-dependency. Good enough for rendering snippets and
// files; not a full-blown parser (no semantic awareness, no generics spans).

export type TokenType =
  | 'kw'
  | 'type'
  | 'str'
  | 'num'
  | 'com'
  | 'id'
  | 'punct'
  | 'ws';

export interface Token {
  type: TokenType;
  text: string;
}

const KEYWORDS = new Set([
  'break', 'case', 'chan', 'const', 'continue', 'default', 'defer', 'else',
  'fallthrough', 'for', 'func', 'go', 'goto', 'if', 'import', 'interface',
  'map', 'package', 'range', 'return', 'select', 'struct', 'switch', 'type',
  'var',
]);

const BUILTINS = new Set([
  'nil', 'true', 'false', 'iota',
  'string', 'int', 'int8', 'int16', 'int32', 'int64',
  'uint', 'uint8', 'uint16', 'uint32', 'uint64', 'uintptr',
  'byte', 'rune', 'float32', 'float64', 'complex64', 'complex128',
  'bool', 'error', 'any', 'comparable',
  'make', 'new', 'len', 'cap', 'append', 'copy', 'delete', 'close',
  'panic', 'recover', 'print', 'println',
]);

const IDENT_START = /[A-Za-z_]/;
const IDENT_CONT = /[A-Za-z0-9_]/;
const DIGIT = /[0-9]/;
const NUM_CONT = /[0-9a-fA-FxXoObB._+\-eE]/;

/**
 * tokenize scans Go source into a flat token stream. Whitespace and newlines
 * are preserved as `ws` tokens so callers can reconstruct the original text
 * by concatenating tokens in order.
 */
export function tokenize(src: string): Token[] {
  const out: Token[] = [];
  let i = 0;
  const n = src.length;

  while (i < n) {
    const c = src[i];

    // Whitespace (including newlines).
    if (c === ' ' || c === '\t' || c === '\n' || c === '\r') {
      let j = i;
      while (j < n && (src[j] === ' ' || src[j] === '\t' || src[j] === '\n' || src[j] === '\r')) j++;
      out.push({ type: 'ws', text: src.slice(i, j) });
      i = j;
      continue;
    }

    // Line comment.
    if (c === '/' && src[i + 1] === '/') {
      let j = i;
      while (j < n && src[j] !== '\n') j++;
      out.push({ type: 'com', text: src.slice(i, j) });
      i = j;
      continue;
    }

    // Block comment.
    if (c === '/' && src[i + 1] === '*') {
      let j = i + 2;
      while (j < n - 1 && !(src[j] === '*' && src[j + 1] === '/')) j++;
      j = Math.min(n, j + 2);
      out.push({ type: 'com', text: src.slice(i, j) });
      i = j;
      continue;
    }

    // Interpreted string literal.
    if (c === '"') {
      let j = i + 1;
      while (j < n && src[j] !== '"' && src[j] !== '\n') {
        if (src[j] === '\\' && j + 1 < n) j++;
        j++;
      }
      j = Math.min(n, j + 1);
      out.push({ type: 'str', text: src.slice(i, j) });
      i = j;
      continue;
    }

    // Raw string (backtick).
    if (c === '`') {
      let j = i + 1;
      while (j < n && src[j] !== '`') j++;
      j = Math.min(n, j + 1);
      out.push({ type: 'str', text: src.slice(i, j) });
      i = j;
      continue;
    }

    // Rune literal.
    if (c === "'") {
      let j = i + 1;
      while (j < n && src[j] !== "'" && src[j] !== '\n') {
        if (src[j] === '\\' && j + 1 < n) j++;
        j++;
      }
      j = Math.min(n, j + 1);
      out.push({ type: 'str', text: src.slice(i, j) });
      i = j;
      continue;
    }

    // Number.
    if (DIGIT.test(c) || (c === '.' && DIGIT.test(src[i + 1] ?? ''))) {
      let j = i + 1;
      while (j < n && NUM_CONT.test(src[j])) j++;
      out.push({ type: 'num', text: src.slice(i, j) });
      i = j;
      continue;
    }

    // Identifier / keyword.
    if (IDENT_START.test(c)) {
      let j = i;
      while (j < n && IDENT_CONT.test(src[j])) j++;
      const word = src.slice(i, j);
      let type: TokenType = 'id';
      if (KEYWORDS.has(word)) type = 'kw';
      else if (BUILTINS.has(word)) type = 'type';
      out.push({ type, text: word });
      i = j;
      continue;
    }

    // Everything else is a single-char punctuation token.
    out.push({ type: 'punct', text: c });
    i++;
  }

  return out;
}

/**
 * tokensByLine splits the token stream into lines, breaking `ws` / `com` / `str`
 * tokens at newline boundaries so each line's token list reconstructs exactly
 * one line of source text.
 */
export function tokensByLine(src: string): Token[][] {
  const lines: Token[][] = [[]];
  for (const tok of tokenize(src)) {
    if (!tok.text.includes('\n')) {
      lines[lines.length - 1].push(tok);
      continue;
    }
    const parts = tok.text.split('\n');
    for (let k = 0; k < parts.length; k++) {
      if (parts[k]) lines[lines.length - 1].push({ type: tok.type, text: parts[k] });
      if (k < parts.length - 1) lines.push([]);
    }
  }
  return lines;
}
