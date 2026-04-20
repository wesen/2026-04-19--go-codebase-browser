import * as ts from 'typescript';
import * as path from 'path';
import * as crypto from 'crypto';
import { fileURLToPath } from 'url';
import { symbolID, methodID, packageID, fileID } from './ids.js';
import type { Index, Package, File, Symbol, Range, Kind } from './types.js';

export interface ExtractOptions {
  moduleRoot: string;
  tsconfig?: string;
  moduleName?: string;
  /**
   * Prefix prepended to every File.path. Used when the TS project is a
   * subdirectory of a larger repo — setting pathPrefix="ui" makes the
   * emitted file paths resolvable against the outer repo's source FS.
   * Does NOT change symbol IDs (those must be stable across repo moves).
   */
  pathPrefix?: string;
}

/** Extract builds an Index for the TypeScript project at options.moduleRoot. */
export function extract(opts: ExtractOptions): Index {
  const root = path.resolve(opts.moduleRoot);
  const tsconfigPath =
    opts.tsconfig ??
    ts.findConfigFile(root, ts.sys.fileExists, 'tsconfig.json');
  if (!tsconfigPath) {
    throw new Error(`no tsconfig.json under ${root}`);
  }
  const absTsconfig = path.isAbsolute(tsconfigPath)
    ? tsconfigPath
    : path.resolve(root, tsconfigPath);
  const raw = ts.sys.readFile(absTsconfig);
  if (!raw) throw new Error(`cannot read ${absTsconfig}`);
  const parsed = ts.parseJsonConfigFileContent(
    JSON.parse(raw),
    ts.sys,
    path.dirname(absTsconfig),
  );
  const program = ts.createProgram({
    rootNames: parsed.fileNames,
    options: parsed.options,
  });

  const moduleName = opts.moduleName ?? path.basename(root);
  const prefix = (opts.pathPrefix ?? '').replace(/\\/g, '/').replace(/\/$/, '');
  const joinPrefix = (p: string) => (prefix ? `${prefix}/${p}` : p);

  const idx: Index = {
    version: '1',
    generatedAt: new Date().toISOString(),
    module: moduleName,
    language: 'ts',
    packages: [],
    files: [],
    symbols: [],
    refs: [],
  };

  const pkgByImportPath = new Map<string, Package>();

  for (const sf of program.getSourceFiles()) {
    if (sf.isDeclarationFile) continue;
    const absFile = sf.fileName;
    if (!absFile.startsWith(root + path.sep) && absFile !== root) continue;
    // Skip anything inside node_modules — tsconfig's default includes may
    // still pull in transitively-referenced .ts files from installed
    // packages, which we don't want to index as "our" source.
    if (absFile.includes(`${path.sep}node_modules${path.sep}`)) continue;
    const relNative = path.relative(root, absFile).replace(/\\/g, '/');
    // File.path is prefix-scoped so it resolves against the outer repo's
    // source FS when this index lives in a Go-module-rooted server.
    const rel = joinPrefix(relNative);

    const dir = path.dirname(relNative).replace(/\\/g, '/');
    const importPath = dir === '.' ? moduleName : `${moduleName}/${dir}`;
    let pkg = pkgByImportPath.get(importPath);
    if (!pkg) {
      pkg = {
        id: packageID(importPath),
        importPath,
        name: path.basename(dir === '.' ? moduleName : dir),
        fileIds: [],
        symbolIds: [],
        language: 'ts',
      };
      pkgByImportPath.set(importPath, pkg);
      idx.packages.push(pkg);
    }

    const bytes = ts.sys.readFile(absFile) ?? '';
    const sha = crypto.createHash('sha256').update(bytes).digest('hex');
    const file: File = {
      id: fileID(rel),
      path: rel,
      packageId: pkg.id,
      size: bytes.length,
      lineCount: bytes === '' ? 0 : bytes.split('\n').length,
      sha256: sha,
      language: 'ts',
    };
    idx.files.push(file);
    pkg.fileIds.push(file.id);

    // TypeScript treats each file as its own module, so symbols in different
    // files inside the same directory (e.g. Storybook's `const meta` in every
    // *.stories.tsx) need file-scoped IDs to stay unique. We keep `pkg.id`
    // as the directory grouping for tree-nav, but the ID segment threaded
    // through symbols uses the relative path minus extension.
    // Symbol scope uses the un-prefixed path: IDs must stay stable across
    // repo-layout changes (moving the ts project from ui/ to web/ shouldn't
    // invalidate every doc snippet referencing a TS symbol).
    const symScope = `${moduleName}/${relNative.replace(/\.(tsx?|mts|cts)$/, '')}`;

    ts.forEachChild(sf, (node) => {
      collectTopLevel(node, sf, symScope, file.id, pkg!, idx);
    });
  }

  // Deterministic sort, matching internal/indexer/extractor.go:sortIndex.
  idx.packages.sort((a, b) => a.importPath.localeCompare(b.importPath));
  idx.files.sort((a, b) => a.path.localeCompare(b.path));
  idx.symbols.sort((a, b) => {
    if (a.packageId !== b.packageId) return a.packageId.localeCompare(b.packageId);
    if (a.fileId !== b.fileId) return a.fileId.localeCompare(b.fileId);
    return a.range.startOffset - b.range.startOffset;
  });
  for (const p of idx.packages) {
    p.fileIds.sort();
    p.symbolIds.sort();
  }
  return idx;
}

function rangeFrom(sf: ts.SourceFile, pos: number, end: number): Range {
  const s = sf.getLineAndCharacterOfPosition(pos);
  const e = sf.getLineAndCharacterOfPosition(end);
  return {
    startLine: s.line + 1,
    startCol: s.character + 1,
    endLine: e.line + 1,
    endCol: e.character + 1,
    startOffset: pos,
    endOffset: end,
  };
}

function isExported(node: ts.Node): boolean {
  return !!(ts.getCombinedModifierFlags(node as ts.Declaration) & ts.ModifierFlags.Export);
}

function signatureText(node: ts.Node, sf: ts.SourceFile): string {
  const src = node.getText(sf);
  const open = src.indexOf('{');
  return (open > 0 ? src.slice(0, open) : src).trim();
}

function jsDoc(node: ts.Node): string | undefined {
  // ts.jsDoc is internal-but-stable. If it ever disappears, fall back to
  // ts.getJSDocCommentsAndTags().
  const tags = (node as unknown as { jsDoc?: ts.JSDoc[] }).jsDoc;
  if (!tags || tags.length === 0) return undefined;
  const parts = tags
    .map((d) => (typeof d.comment === 'string' ? d.comment.trim() : ''))
    .filter(Boolean);
  return parts.length ? parts.join('\n') : undefined;
}

function addSymbol(idx: Index, pkg: Package, s: Symbol) {
  idx.symbols.push(s);
  pkg.symbolIds.push(s.id);
}

function collectTopLevel(
  node: ts.Node,
  sf: ts.SourceFile,
  importPath: string,
  fid: string,
  pkg: Package,
  idx: Index,
) {
  if (ts.isFunctionDeclaration(node) && node.name) {
    const name = node.name.text;
    addSymbol(idx, pkg, {
      id: symbolID(importPath, 'func', name),
      kind: 'func',
      name,
      packageId: pkg.id,
      fileId: fid,
      range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
      signature: signatureText(node, sf),
      doc: jsDoc(node),
      exported: isExported(node),
      language: 'ts',
    });
  } else if (ts.isClassDeclaration(node) && node.name) {
    const className = node.name.text;
    const exported = isExported(node);
    addSymbol(idx, pkg, {
      id: symbolID(importPath, 'class', className),
      kind: 'class',
      name: className,
      packageId: pkg.id,
      fileId: fid,
      range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
      signature: `class ${className}`,
      doc: jsDoc(node),
      exported,
      language: 'ts',
    });
    for (const m of node.members) {
      if (ts.isMethodDeclaration(m) && m.name && ts.isIdentifier(m.name)) {
        const mname = m.name.text;
        addSymbol(idx, pkg, {
          id: methodID(importPath, className, mname),
          kind: 'method',
          name: mname,
          packageId: pkg.id,
          fileId: fid,
          range: rangeFrom(sf, m.getStart(sf), m.getEnd()),
          signature: signatureText(m, sf),
          doc: jsDoc(m),
          receiver: { typeName: className, pointer: false },
          exported,
          language: 'ts',
        });
      }
    }
  } else if (ts.isInterfaceDeclaration(node)) {
    const name = node.name.text;
    addSymbol(idx, pkg, {
      id: symbolID(importPath, 'iface', name),
      kind: 'iface',
      name,
      packageId: pkg.id,
      fileId: fid,
      range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
      signature: `interface ${name}`,
      doc: jsDoc(node),
      exported: isExported(node),
      language: 'ts',
    });
  } else if (ts.isTypeAliasDeclaration(node)) {
    const name = node.name.text;
    addSymbol(idx, pkg, {
      id: symbolID(importPath, 'alias', name),
      kind: 'alias',
      name,
      packageId: pkg.id,
      fileId: fid,
      range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
      signature: `type ${name}`,
      doc: jsDoc(node),
      exported: isExported(node),
      language: 'ts',
    });
  } else if (ts.isVariableStatement(node)) {
    const kind: Kind =
      node.declarationList.flags & ts.NodeFlags.Const ? 'const' : 'var';
    const exported = !!(
      ts.getCombinedModifierFlags(
        node.declarationList.declarations[0] as ts.Declaration,
      ) & ts.ModifierFlags.Export
    );
    for (const d of node.declarationList.declarations) {
      if (!ts.isIdentifier(d.name)) continue;
      const name = d.name.text;
      addSymbol(idx, pkg, {
        id: symbolID(importPath, kind, name),
        kind,
        name,
        packageId: pkg.id,
        fileId: fid,
        range: rangeFrom(sf, node.getStart(sf), node.getEnd()),
        signature: d.getText(sf),
        doc: jsDoc(node),
        exported,
        language: 'ts',
      });
    }
  }
}

// Let this module be runnable as a script for quick iteration:
// `tsx src/extract.ts <moduleRoot>` — prints the Index to stdout.
const thisFile = fileURLToPath(import.meta.url);
if (process.argv[1] === thisFile) {
  const idx = extract({ moduleRoot: process.argv[2] ?? '.' });
  process.stdout.write(JSON.stringify(idx, null, 2) + '\n');
}
