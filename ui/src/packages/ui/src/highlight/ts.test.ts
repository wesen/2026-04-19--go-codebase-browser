// Lightweight smoke assertions for the TS tokenizer. Mirrors highlight/go.test.ts —
// runnable via `tsx highlight/ts.test.ts` for quick feedback.
import { tokenize } from './ts';

function assert(cond: unknown, msg: string) {
  if (!cond) throw new Error('assert: ' + msg);
}

function roundtrip(src: string) {
  const joined = tokenize(src).map((t) => t.text).join('');
  assert(joined === src, `roundtrip mismatch for: ${JSON.stringify(src)}`);
}

roundtrip('export function greet(name: string): string { return name }');
roundtrip("const s: string = 'hello';");
roundtrip('const t = `hello ${name}, you have ${count} items`;');
roundtrip('// single line\nconst x = 1;');
roundtrip('/* multi\nline */ const y = 2;');
roundtrip('interface Props { name: string; count: number }');
roundtrip('function View() { return <div className="c">hi</div>; }');

// Capitalized identifiers immediately after `<` should tokenize as JSX
// components (color role = "type").
const comp = tokenize('<Button/>');
const btn = comp.find((t) => t.text === 'Button');
assert(btn?.type === 'type', 'Button after < is type');

// Lowercase DOM tags keep the default id color.
const dom = tokenize('<div />');
const div = dom.find((t) => t.text === 'div');
assert(div?.type === 'id', 'div after < stays id');

// Closing tags also take effect: </Button>.
const close = tokenize('</Button>');
const cBtn = close.find((t) => t.text === 'Button');
assert(cBtn?.type === 'type', 'Button after </ is type');

// Comparison with whitespace must not trigger the JSX heuristic.
const cmp = tokenize('a < B');
const B = cmp.find((t) => t.text === 'B');
assert(B?.type === 'id', 'B after "< " is id (not JSX)');

// eslint-disable-next-line no-console
console.log('ok');
