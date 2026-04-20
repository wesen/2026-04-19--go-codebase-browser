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
  // Populated during the symbol pass, consumed during the refs pass.
  // Maps every TS declaration node we emitted a Symbol for to that
  // symbol's ID, so TypeChecker-resolved identifier uses can be rewritten
  // into inter-symbol Ref records.
  const declToSymbolID = new Map<ts.Declaration, string>();
  // File IDs we emitted symbols for, keyed by absolute filename — the
  // refs pass needs the same fid for Ref.fileId.
  const fileIDByAbs = new Map<string, string>();

  const projectFiles = program.getSourceFiles().filter((sf) => {
    if (sf.isDeclarationFile) return false;
    const absFile = sf.fileName;
    if (!absFile.startsWith(root + path.sep) && absFile !== root) return false;
    if (absFile.includes(`${path.sep}node_modules${path.sep}`)) return false;
    return true;
  });

  for (const sf of projectFiles) {
    const absFile = sf.fileName;
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
    fileIDByAbs.set(absFile, file.id);

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
      collectTopLevel(node, sf, symScope, file.id, pkg!, idx, declToSymbolID);
    });
  }

  // Refs pass: walk function/method bodies and resolve identifier uses
  // against declToSymbolID via the TypeChecker. Runs after symbol extraction
  // so every decl we might point at is already registered.
  const checker = program.getTypeChecker();
  for (const sf of projectFiles) {
    const fid = fileIDByAbs.get(sf.fileName);
    if (!fid) continue;
    ts.forEachChild(sf, (node) => {
      collectRefs(node, sf, fid, declToSymbolID, checker, idx);
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
  idx.refs.sort((a, b) => {
    if (a.fileId !== b.fileId) return a.fileId.localeCompare(b.fileId);
    if (a.range.startOffset !== b.range.startOffset) {
      return a.range.startOffset - b.range.startOffset;
    }
    if (a.fromSymbolId !== b.fromSymbolId) {
      return a.fromSymbolId.localeCompare(b.fromSymbolId);
    }
    return a.toSymbolId.localeCompare(b.toSymbolId);
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
    // TS positions are UTF-16 code units; the Go server slices source bytes
    // in UTF-8. Convert up-front so all downstream consumers (snippet,
    // snippet-refs, xref) stay byte-aligned for files that contain
    // non-ASCII characters (e.g. the HORIZONTAL ELLIPSIS in DocPage.tsx).
    startOffset: utf16ToByte(sf, pos),
    endOffset: utf16ToByte(sf, end),
  };
}

// Cached UTF-16-position → UTF-8-byte-offset table, computed once per file.
// For ASCII-only files this is the identity; for files with non-BMP/multi-
// byte chars it accumulates the extra bytes per TS position so the reported
// offsets match what the Go server sees when it reads the file as bytes.
const utf16ToByteCache = new WeakMap<ts.SourceFile, number[]>();

function utf16ToByte(sf: ts.SourceFile, pos: number): number {
  let offsets = utf16ToByteCache.get(sf);
  if (!offsets) {
    const text = sf.text;
    offsets = new Array<number>(text.length + 1);
    let byte = 0;
    for (let i = 0; i < text.length; i++) {
      offsets[i] = byte;
      const code = text.charCodeAt(i);
      if (code < 0x80) byte += 1;
      else if (code < 0x800) byte += 2;
      else if (code >= 0xd800 && code <= 0xdbff) byte += 4; // high surrogate pair
      else if (code >= 0xdc00 && code <= 0xdfff) byte += 0; // low surrogate (counted above)
      else byte += 3;
    }
    offsets[text.length] = byte;
    utf16ToByteCache.set(sf, offsets);
  }
  return offsets[Math.min(Math.max(pos, 0), offsets.length - 1)];
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
  declToSymbolID: Map<ts.Declaration, string>,
) {
  if (ts.isFunctionDeclaration(node) && node.name) {
    const name = node.name.text;
    const id = symbolID(importPath, 'func', name);
    addSymbol(idx, pkg, {
      id,
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
    declToSymbolID.set(node, id);
  } else if (ts.isClassDeclaration(node) && node.name) {
    const className = node.name.text;
    const exported = isExported(node);
    const classId = symbolID(importPath, 'class', className);
    addSymbol(idx, pkg, {
      id: classId,
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
    declToSymbolID.set(node, classId);
    for (const m of node.members) {
      if (ts.isMethodDeclaration(m) && m.name && ts.isIdentifier(m.name)) {
        const mname = m.name.text;
        const mid = methodID(importPath, className, mname);
        addSymbol(idx, pkg, {
          id: mid,
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
        declToSymbolID.set(m, mid);
      }
    }
  } else if (ts.isInterfaceDeclaration(node)) {
    const name = node.name.text;
    const id = symbolID(importPath, 'iface', name);
    addSymbol(idx, pkg, {
      id,
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
    declToSymbolID.set(node, id);
  } else if (ts.isTypeAliasDeclaration(node)) {
    const name = node.name.text;
    const id = symbolID(importPath, 'alias', name);
    addSymbol(idx, pkg, {
      id,
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
    declToSymbolID.set(node, id);
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
      const id = symbolID(importPath, kind, name);
      addSymbol(idx, pkg, {
        id,
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
      // The checker returns `d` (VariableDeclaration) as the decl for
      // identifier uses of the variable, not the containing statement.
      declToSymbolID.set(d, id);
    }
  }
}

// collectRefs walks function/method bodies and emits a Ref per identifier
// use that resolves (via the TypeChecker) to a declaration we indexed.
// Mirrors internal/indexer/xref.go:addRefsForFile — the top-level traversal
// is intentionally shallow (FunctionDeclaration body + ClassDeclaration
// method bodies), since those are the only places Go emits refs from today.
function collectRefs(
  node: ts.Node,
  sf: ts.SourceFile,
  fid: string,
  declToSymbolID: Map<ts.Declaration, string>,
  checker: ts.TypeChecker,
  idx: Index,
) {
  if (ts.isFunctionDeclaration(node) && node.name && node.body) {
    const fromID = declToSymbolID.get(node);
    if (fromID) {
      emitBodyRefs(node.body, sf, fid, fromID, declToSymbolID, checker, idx);
    }
  } else if (ts.isClassDeclaration(node)) {
    for (const m of node.members) {
      if (ts.isMethodDeclaration(m) && m.body) {
        const fromID = declToSymbolID.get(m);
        if (fromID) {
          emitBodyRefs(m.body, sf, fid, fromID, declToSymbolID, checker, idx);
        }
      }
    }
  }
}

function emitBodyRefs(
  body: ts.Node,
  sf: ts.SourceFile,
  fid: string,
  fromID: string,
  declToSymbolID: Map<ts.Declaration, string>,
  checker: ts.TypeChecker,
  idx: Index,
) {
  const visit = (n: ts.Node): void => {
    if (ts.isIdentifier(n)) {
      // Skip identifiers in binding positions — getSymbolAtLocation still
      // returns the symbol for parameter/local names, but emitting refs
      // from a function to its own parameters would be noise.
      const parent = n.parent;
      if (
        parent &&
        ((ts.isParameter(parent) && parent.name === n) ||
          (ts.isVariableDeclaration(parent) && parent.name === n) ||
          (ts.isBindingElement(parent) && parent.name === n))
      ) {
        ts.forEachChild(n, visit);
        return;
      }
      let sym = checker.getSymbolAtLocation(n);
      // Named imports resolve to an alias symbol pointing at the ImportSpecifier.
      // Follow it to the real declaration so the ref targets the exported
      // symbol (function/class/const) rather than the local import binding.
      if (sym && sym.flags & ts.SymbolFlags.Alias) {
        try {
          sym = checker.getAliasedSymbol(sym);
        } catch {
          // getAliasedSymbol throws on unresolvable aliases; fall back to the
          // original symbol (will just fail the decl-map lookup below).
        }
      }
      const toID = resolveSymbolID(sym, declToSymbolID);
      if (toID && toID !== fromID) {
        idx.refs.push({
          fromSymbolId: fromID,
          toSymbolId: toID,
          kind: refKindFor(sym!),
          fileId: fid,
          range: rangeFrom(sf, n.getStart(sf), n.getEnd()),
        });
      }
    }
    ts.forEachChild(n, visit);
  };
  visit(body);
}

function resolveSymbolID(
  sym: ts.Symbol | undefined,
  declToSymbolID: Map<ts.Declaration, string>,
): string | undefined {
  if (!sym || !sym.declarations) return undefined;
  for (const d of sym.declarations) {
    const id = declToSymbolID.get(d);
    if (id) return id;
  }
  return undefined;
}

function refKindFor(sym: ts.Symbol): string {
  const f = sym.flags;
  if (f & (ts.SymbolFlags.Function | ts.SymbolFlags.Method)) return 'call';
  if (
    f &
    (ts.SymbolFlags.Class |
      ts.SymbolFlags.Interface |
      ts.SymbolFlags.TypeAlias |
      ts.SymbolFlags.Enum)
  ) {
    return 'uses-type';
  }
  if (f & (ts.SymbolFlags.Variable | ts.SymbolFlags.BlockScopedVariable)) {
    return 'reads';
  }
  return 'use';
}

// Let this module be runnable as a script for quick iteration:
// `tsx src/extract.ts <moduleRoot>` — prints the Index to stdout.
const thisFile = fileURLToPath(import.meta.url);
if (process.argv[1] === thisFile) {
  const idx = extract({ moduleRoot: process.argv[2] ?? '.' });
  process.stdout.write(JSON.stringify(idx, null, 2) + '\n');
}
