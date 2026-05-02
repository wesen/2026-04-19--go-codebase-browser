import { describe, expect, it } from 'vitest';

import { tokenize } from './ts';

function expectRoundtrip(src: string) {
  const joined = tokenize(src).map((t) => t.text).join('');
  expect(joined).toBe(src);
}

describe('TypeScript tokenizer', () => {
  it('roundtrips common TS and TSX snippets', () => {
    expectRoundtrip('export function greet(name: string): string { return name }');
    expectRoundtrip("const s: string = 'hello';");
    expectRoundtrip('const t = `hello ${name}, you have ${count} items`;');
    expectRoundtrip('// single line\nconst x = 1;');
    expectRoundtrip('/* multi\nline */ const y = 2;');
    expectRoundtrip('interface Props { name: string; count: number }');
    expectRoundtrip('function View() { return <div className="c">hi</div>; }');
  });

  it('classifies JSX component names without treating comparisons as JSX', () => {
    const comp = tokenize('<Button/>');
    expect(comp.find((t) => t.text === 'Button')?.type).toBe('type');

    const dom = tokenize('<div />');
    expect(dom.find((t) => t.text === 'div')?.type).toBe('id');

    const close = tokenize('</Button>');
    expect(close.find((t) => t.text === 'Button')?.type).toBe('type');

    const cmp = tokenize('a < B');
    expect(cmp.find((t) => t.text === 'B')?.type).toBe('id');
  });
});
