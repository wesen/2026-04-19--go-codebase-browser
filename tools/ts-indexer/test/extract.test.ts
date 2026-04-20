import { describe, it, expect } from 'vitest';
import * as path from 'path';
import { fileURLToPath } from 'url';
import { extract } from '../src/extract.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const FIXTURE = path.resolve(__dirname, 'fixture');

describe('extract', () => {
  const idx = extract({ moduleRoot: FIXTURE, moduleName: 'fixture' });

  it('stamps language=ts on every record', () => {
    for (const p of idx.packages) expect(p.language).toBe('ts');
    for (const f of idx.files) expect(f.language).toBe('ts');
    for (const s of idx.symbols) expect(s.language).toBe('ts');
  });

  it('emits exactly one package (src)', () => {
    expect(idx.packages).toHaveLength(1);
    expect(idx.packages[0].importPath).toBe('fixture/src');
  });

  it('emits symbols covering all phase-1 kinds', () => {
    const byName = Object.fromEntries(idx.symbols.map((s) => [s.name, s.kind]));
    expect(byName).toMatchObject({
      Greeter: 'class',
      hello: 'method',
      MaxRetries: 'const',
      greet: 'func',
      Greetable: 'iface',
      Prefix: 'alias',
      run: 'func',
    });
    // 7 symbols total: 5 top-level + 1 method in greeter.ts, plus run() in main.ts.
    expect(idx.symbols).toHaveLength(7);
  });

  it('method IDs embed the class as receiver and the file-stem as scope', () => {
    const hello = idx.symbols.find((s) => s.name === 'hello');
    // File-scoped: the `src/greeter` segment disambiguates symbols that would
    // otherwise collide across files in the same directory (e.g. Storybook's
    // `const meta` in every *.stories.tsx).
    expect(hello?.id).toBe('sym:fixture/src/greeter.method.Greeter.hello');
    expect(hello?.receiver).toMatchObject({ typeName: 'Greeter', pointer: false });
  });

  it('byte offsets round-trip: slicing source by offset gives the declaration', () => {
    const greet = idx.symbols.find((s) => s.name === 'greet');
    expect(greet).toBeDefined();
    const src = require('fs').readFileSync(
      path.resolve(FIXTURE, greet!.fileId.replace(/^file:/, '')),
      'utf8',
    );
    const snippet = src.slice(greet!.range.startOffset, greet!.range.endOffset);
    expect(snippet).toContain('function greet');
    expect(snippet).toContain('return `Hello, ${name}!`;');
  });

  it('is deterministic across runs', () => {
    const again = extract({ moduleRoot: FIXTURE, moduleName: 'fixture' });
    expect(idx.symbols.map((s) => s.id)).toEqual(again.symbols.map((s) => s.id));
    expect(idx.files.map((f) => f.sha256)).toEqual(again.files.map((f) => f.sha256));
    expect(idx.refs.map((r) => `${r.fromSymbolId}->${r.toSymbolId}`)).toEqual(
      again.refs.map((r) => `${r.fromSymbolId}->${r.toSymbolId}`),
    );
  });

  it('emits refs from run() to greet, Greeter, hello, MaxRetries', () => {
    const runID = 'sym:fixture/src/main.func.run';
    const fromRun = idx.refs.filter((r) => r.fromSymbolId === runID);
    const targets = new Set(fromRun.map((r) => r.toSymbolId));
    expect(targets).toContain('sym:fixture/src/greeter.func.greet');
    expect(targets).toContain('sym:fixture/src/greeter.class.Greeter');
    expect(targets).toContain('sym:fixture/src/greeter.method.Greeter.hello');
    expect(targets).toContain('sym:fixture/src/greeter.const.MaxRetries');
  });

  it('annotates ref kinds', () => {
    const kindByTarget = new Map<string, string>();
    for (const r of idx.refs) kindByTarget.set(r.toSymbolId, r.kind);
    expect(kindByTarget.get('sym:fixture/src/greeter.func.greet')).toBe('call');
    expect(kindByTarget.get('sym:fixture/src/greeter.method.Greeter.hello')).toBe('call');
    expect(kindByTarget.get('sym:fixture/src/greeter.class.Greeter')).toBe('uses-type');
    expect(kindByTarget.get('sym:fixture/src/greeter.const.MaxRetries')).toBe('reads');
  });
});
