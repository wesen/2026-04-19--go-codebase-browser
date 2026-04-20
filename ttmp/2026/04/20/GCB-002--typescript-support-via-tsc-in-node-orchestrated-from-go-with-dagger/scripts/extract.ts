// Prototype TypeScript extractor that emits records shaped like the Go
// indexer's Index JSON (packages/files/symbols/refs). Use from this dir:
//
//   npx -p typescript@5 -p tsx tsx extract.ts fixture-ts
//
// Output is written to stdout as pretty JSON. This file is deliberately
// self-contained — no dependencies on the main codebase-browser tree — so
// it can be run in isolation during design validation.

import * as ts from 'typescript';
import * as path from 'path';
import * as fs from 'fs';
import * as crypto from 'crypto';

type Kind = 'func' | 'method' | 'class' | 'iface' | 'type' | 'alias' | 'const' | 'var';

interface Range {
  startLine: number; startCol: number; endLine: number; endCol: number;
  startOffset: number; endOffset: number;
}
interface Sym {
  id: string; kind: Kind; name: string; packageId: string; fileId: string;
  range: Range; doc?: string; signature?: string; exported: boolean;
  language: 'ts';
}
interface File {
  id: string; path: string; packageId: string; size: number; lineCount: number;
  sha256: string; language: 'ts';
}
interface Pkg { id: string; importPath: string; name: string; fileIds: string[]; symbolIds: string[]; language: 'ts'; }
interface Ref { fromSymbolId: string; toSymbolId: string; kind: string; fileId: string; range: Range; }

interface Index {
  version: string; generatedAt: string; module: string; language: 'ts';
  packages: Pkg[]; files: File[]; symbols: Sym[]; refs: Ref[];
}

function rangeFrom(sf: ts.SourceFile, pos: number, end: number): Range {
  const start = sf.getLineAndCharacterOfPosition(pos);
  const stop = sf.getLineAndCharacterOfPosition(end);
  return {
    startLine: start.line + 1, startCol: start.character + 1,
    endLine: stop.line + 1, endCol: stop.character + 1,
    startOffset: pos, endOffset: end,
  };
}

function symbolID(importPath: string, kind: Kind, name: string): string {
  return `sym:${importPath}.${kind}.${name}`;
}

function methodID(importPath: string, recv: string, name: string): string {
  return `sym:${importPath}.method.${recv}.${name}`;
}

function packageIDOf(importPath: string): string { return `pkg:${importPath}`; }
function fileIDOf(relPath: string): string { return `file:${relPath}`; }

function main() {
  const root = path.resolve(process.argv[2] ?? '.');
  const configPath = ts.findConfigFile(root, ts.sys.fileExists, 'tsconfig.json');
  if (!configPath) {
    console.error(`no tsconfig.json found under ${root}`);
    process.exit(1);
  }
  const parsed = ts.parseJsonConfigFileContent(
    JSON.parse(ts.sys.readFile(configPath) ?? '{}'),
    ts.sys,
    path.dirname(configPath),
  );
  const program = ts.createProgram({ rootNames: parsed.fileNames, options: parsed.options });
  const checker = program.getTypeChecker();

  const idx: Index = {
    version: '1',
    generatedAt: new Date().toISOString(),
    module: path.basename(root),
    language: 'ts',
    packages: [],
    files: [],
    symbols: [],
    refs: [],
  };

  const pkgById = new Map<string, Pkg>();

  for (const sf of program.getSourceFiles()) {
    if (sf.isDeclarationFile) continue;
    const abs = sf.fileName;
    if (!abs.startsWith(root)) continue;
    const rel = path.relative(root, abs);

    // Package = directory (relative to root).
    const dir = path.dirname(rel).replace(/\\/g, '/');
    const importPath = (idx.module + '/' + dir).replace(/\/\.$/, '');
    let pkg = pkgById.get(importPath);
    if (!pkg) {
      pkg = {
        id: packageIDOf(importPath),
        importPath,
        name: path.basename(dir === '.' ? idx.module : dir),
        fileIds: [],
        symbolIds: [],
        language: 'ts',
      };
      pkgById.set(importPath, pkg);
      idx.packages.push(pkg);
    }

    const bytes = ts.sys.readFile(abs) ?? '';
    const sha = crypto.createHash('sha256').update(bytes).digest('hex');
    const file: File = {
      id: fileIDOf(rel),
      path: rel,
      packageId: pkg.id,
      size: bytes.length,
      lineCount: bytes.split('\n').length,
      sha256: sha,
      language: 'ts',
    };
    idx.files.push(file);
    pkg.fileIds.push(file.id);

    ts.forEachChild(sf, (node) => visitTopLevel(node, sf, importPath, file.id, pkg!, idx, checker));
  }

  // Stable ordering for deterministic output.
  idx.packages.sort((a, b) => a.importPath.localeCompare(b.importPath));
  idx.files.sort((a, b) => a.path.localeCompare(b.path));
  idx.symbols.sort((a, b) =>
    a.packageId !== b.packageId ? a.packageId.localeCompare(b.packageId)
    : a.fileId !== b.fileId ? a.fileId.localeCompare(b.fileId)
    : a.range.startOffset - b.range.startOffset);
  for (const p of idx.packages) {
    p.fileIds.sort();
    p.symbolIds.sort();
  }

  process.stdout.write(JSON.stringify(idx, null, 2) + '\n');
}

function visitTopLevel(
  node: ts.Node, sf: ts.SourceFile, importPath: string, fileId: string,
  pkg: Pkg, idx: Index, checker: ts.TypeChecker,
) {
  if (ts.isFunctionDeclaration(node) && node.name) {
    const name = node.name.text;
    const sig = getSignatureText(node, sf);
    const sym: Sym = {
      id: symbolID(importPath, 'func', name),
      kind: 'func', name,
      packageId: pkg.id, fileId,
      range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
      signature: sig,
      doc: getJsDoc(node),
      exported: !!(ts.getCombinedModifierFlags(node) & ts.ModifierFlags.Export),
      language: 'ts',
    };
    idx.symbols.push(sym); pkg.symbolIds.push(sym.id);
  } else if (ts.isClassDeclaration(node) && node.name) {
    const name = node.name.text;
    const sym: Sym = {
      id: symbolID(importPath, 'class', name),
      kind: 'class', name,
      packageId: pkg.id, fileId,
      range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
      doc: getJsDoc(node),
      signature: `class ${name}`,
      exported: !!(ts.getCombinedModifierFlags(node) & ts.ModifierFlags.Export),
      language: 'ts',
    };
    idx.symbols.push(sym); pkg.symbolIds.push(sym.id);
    // Methods.
    for (const m of node.members) {
      if (ts.isMethodDeclaration(m) && m.name && ts.isIdentifier(m.name)) {
        const mname = m.name.text;
        const msym: Sym = {
          id: methodID(importPath, name, mname),
          kind: 'method', name: mname,
          packageId: pkg.id, fileId,
          range: rangeFrom(sf, m.getStart(sf), m.getEnd()),
          signature: getSignatureText(m, sf),
          doc: getJsDoc(m),
          exported: sym.exported,
          language: 'ts',
        };
        idx.symbols.push(msym); pkg.symbolIds.push(msym.id);
      }
    }
  } else if (ts.isInterfaceDeclaration(node)) {
    const name = node.name.text;
    const sym: Sym = {
      id: symbolID(importPath, 'iface', name),
      kind: 'iface', name,
      packageId: pkg.id, fileId,
      range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
      signature: `interface ${name}`,
      doc: getJsDoc(node),
      exported: !!(ts.getCombinedModifierFlags(node) & ts.ModifierFlags.Export),
      language: 'ts',
    };
    idx.symbols.push(sym); pkg.symbolIds.push(sym.id);
  } else if (ts.isTypeAliasDeclaration(node)) {
    const name = node.name.text;
    const sym: Sym = {
      id: symbolID(importPath, 'alias', name),
      kind: 'alias', name,
      packageId: pkg.id, fileId,
      range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
      signature: `type ${name}`,
      doc: getJsDoc(node),
      exported: !!(ts.getCombinedModifierFlags(node) & ts.ModifierFlags.Export),
      language: 'ts',
    };
    idx.symbols.push(sym); pkg.symbolIds.push(sym.id);
  } else if (ts.isVariableStatement(node)) {
    const kind: Kind = (node.declarationList.flags & ts.NodeFlags.Const) ? 'const' : 'var';
    for (const d of node.declarationList.declarations) {
      if (!ts.isIdentifier(d.name)) continue;
      const name = d.name.text;
      const sym: Sym = {
        id: symbolID(importPath, kind, name),
        kind, name,
        packageId: pkg.id, fileId,
        range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
        signature: d.getText(sf),
        doc: getJsDoc(node),
        exported: !!(ts.getCombinedModifierFlags(node) & ts.ModifierFlags.Export),
        language: 'ts',
      };
      idx.symbols.push(sym); pkg.symbolIds.push(sym.id);
    }
  }
}

function getSignatureText(node: ts.Node, sf: ts.SourceFile): string {
  // Signature only (no body): find first '{' at current brace depth and slice.
  const src = node.getText(sf);
  const open = src.indexOf('{');
  return (open > 0 ? src.slice(0, open) : src).trim();
}

function getJsDoc(node: ts.Node): string | undefined {
  const jsDocs = (node as any).jsDoc as ts.JSDoc[] | undefined;
  if (!jsDocs || jsDocs.length === 0) return undefined;
  return jsDocs.map((d) => (d.comment ? String(d.comment).trim() : '')).filter(Boolean).join('\n');
}

main();
