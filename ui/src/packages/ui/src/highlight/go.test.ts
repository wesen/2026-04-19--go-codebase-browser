// Lightweight smoke assertions for the tokenizer. We don't have vitest wired
// yet; this file is exported so a future test runner picks it up. Invoking
// it via `tsx` or `node --import=tsx` gives immediate feedback during dev.
import { tokenize, tokensByLine } from './go';

function assert(cond: unknown, msg: string) {
  if (!cond) throw new Error('assert: ' + msg);
}

function roundtrip(src: string) {
  const joined = tokenize(src).map((t) => t.text).join('');
  assert(joined === src, `roundtrip mismatch for: ${JSON.stringify(src)}`);
}

roundtrip('package foo\n\nfunc Greet(name string) string { return name }');
roundtrip('// Hello\npackage foo');
roundtrip('var s = "hello\\tworld"');
roundtrip('const raw = `line1\nline2`');
roundtrip("const r = 'x'");
roundtrip('var n = 1_000_000');
roundtrip('var f = 3.14e-9');
roundtrip('/* block\ncomment */ var x = 1');

const kwTokens = tokenize('func').map((t) => t.type);
assert(kwTokens[0] === 'kw', 'func is keyword');

const typeTokens = tokenize('string').map((t) => t.type);
assert(typeTokens[0] === 'type', 'string is builtin type');

const lines = tokensByLine('a\nb\nc');
assert(lines.length === 3, 'tokensByLine 3 lines');

// eslint-disable-next-line no-console
console.log('ok');
