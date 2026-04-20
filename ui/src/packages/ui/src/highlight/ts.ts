// Minimal TypeScript/JSX source tokenizer. Mirrors highlight/go.ts —
// same Token shape, same data-tok labels — so <Code> + <SourceView> can
// dispatch by language without changing their rendering logic.

import type { Token, TokenType } from './go';

const KEYWORDS = new Set([
  // Control flow / declarations.
  'break', 'case', 'catch', 'class', 'const', 'continue', 'debugger', 'default',
  'delete', 'do', 'else', 'enum', 'export', 'extends', 'finally', 'for',
  'from', 'function', 'if', 'implements', 'import', 'in', 'instanceof',
  'interface', 'let', 'module', 'namespace', 'new', 'of', 'package',
  'private', 'protected', 'public', 'readonly', 'return', 'super',
  'switch', 'this', 'throw', 'try', 'type', 'typeof', 'var', 'void',
  'while', 'with', 'yield',
  // Modifiers unique to TS.
  'abstract', 'as', 'async', 'await', 'declare', 'is', 'keyof', 'override',
  'satisfies', 'static', 'unique',
]);

const BUILTINS = new Set([
  // Primitives / types.
  'any', 'bigint', 'boolean', 'never', 'null', 'number', 'object', 'string',
  'symbol', 'undefined', 'unknown',
  // Stdlib types most commonly referenced inline.
  'Array', 'Boolean', 'Date', 'Error', 'Function', 'Map', 'Number', 'Object',
  'Promise', 'ReadonlyArray', 'Record', 'RegExp', 'Set', 'String', 'Symbol',
  'WeakMap', 'WeakSet',
  // Literals.
  'true', 'false',
]);

const IDENT_START = /[A-Za-z_$]/;
const IDENT_CONT = /[A-Za-z0-9_$]/;
const DIGIT = /[0-9]/;
const NUM_CONT = /[0-9a-fA-FxXoObBn._+\-eE]/;

/** tokenize scans TS source into a flat token stream. */
export function tokenize(src: string): Token[] {
  const out: Token[] = [];
  let i = 0;
  const n = src.length;

  while (i < n) {
    const c = src[i];

    if (c === ' ' || c === '\t' || c === '\n' || c === '\r') {
      let j = i;
      while (j < n && (src[j] === ' ' || src[j] === '\t' || src[j] === '\n' || src[j] === '\r')) j++;
      out.push({ type: 'ws', text: src.slice(i, j) });
      i = j;
      continue;
    }

    if (c === '/' && src[i + 1] === '/') {
      let j = i;
      while (j < n && src[j] !== '\n') j++;
      out.push({ type: 'com', text: src.slice(i, j) });
      i = j;
      continue;
    }

    if (c === '/' && src[i + 1] === '*') {
      let j = i + 2;
      while (j < n - 1 && !(src[j] === '*' && src[j + 1] === '/')) j++;
      j = Math.min(n, j + 2);
      out.push({ type: 'com', text: src.slice(i, j) });
      i = j;
      continue;
    }

    // Double-quoted string.
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

    // Single-quoted string.
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

    // Template literal (backtick). Phase-1 keeps interpolation inside the
    // string token; full ${...} lexing is out of scope.
    if (c === '`') {
      let j = i + 1;
      let depth = 0;
      while (j < n) {
        if (src[j] === '\\' && j + 1 < n) { j += 2; continue; }
        if (src[j] === '$' && src[j + 1] === '{') { depth++; j += 2; continue; }
        if (depth > 0 && src[j] === '}') { depth--; j++; continue; }
        if (depth === 0 && src[j] === '`') { j++; break; }
        j++;
      }
      out.push({ type: 'str', text: src.slice(i, j) });
      i = j;
      continue;
    }

    if (DIGIT.test(c) || (c === '.' && DIGIT.test(src[i + 1] ?? ''))) {
      let j = i + 1;
      while (j < n && NUM_CONT.test(src[j])) j++;
      out.push({ type: 'num', text: src.slice(i, j) });
      i = j;
      continue;
    }

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

    out.push({ type: 'punct', text: c });
    i++;
  }

  return out;
}

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
