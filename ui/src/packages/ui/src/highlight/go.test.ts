import { describe, expect, it } from 'vitest';

import { tokenize, tokensByLine } from './go';

function expectRoundtrip(src: string) {
  const joined = tokenize(src).map((t) => t.text).join('');
  expect(joined).toBe(src);
}

describe('Go tokenizer', () => {
  it('roundtrips common Go snippets', () => {
    expectRoundtrip('package foo\n\nfunc Greet(name string) string { return name }');
    expectRoundtrip('// Hello\npackage foo');
    expectRoundtrip('var s = "hello\\tworld"');
    expectRoundtrip('const raw = `line1\nline2`');
    expectRoundtrip("const r = 'x'");
    expectRoundtrip('var n = 1_000_000');
    expectRoundtrip('var f = 3.14e-9');
    expectRoundtrip('/* block\ncomment */ var x = 1');
  });

  it('classifies keywords and builtins', () => {
    expect(tokenize('func').map((t) => t.type)[0]).toBe('kw');
    expect(tokenize('string').map((t) => t.type)[0]).toBe('type');
  });

  it('groups tokens by source line', () => {
    expect(tokensByLine('a\nb\nc')).toHaveLength(3);
  });
});
