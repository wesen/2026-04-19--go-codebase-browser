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
    });
    // 6 symbols total: 5 top-level + 1 method.
    expect(idx.symbols).toHaveLength(6);
  });

  it('method IDs embed the class as receiver', () => {
    const hello = idx.symbols.find((s) => s.name === 'hello');
    expect(hello?.id).toBe('sym:fixture/src.method.Greeter.hello');
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
  });
});
