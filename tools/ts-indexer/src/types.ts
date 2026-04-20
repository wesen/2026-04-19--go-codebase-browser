// Mirror of internal/indexer/types.go. Keep the two in sync manually —
// they're small and a codegen dep would be overkill.

export type Kind =
  | 'func'
  | 'method'
  | 'class'
  | 'iface'
  | 'struct'
  | 'type'
  | 'alias'
  | 'const'
  | 'var';

export interface Range {
  startLine: number;
  startCol: number;
  endLine: number;
  endCol: number;
  startOffset: number;
  endOffset: number;
}

export interface Receiver {
  typeName: string;
  pointer: boolean;
}

export interface Symbol {
  id: string;
  kind: Kind;
  name: string;
  packageId: string;
  fileId: string;
  range: Range;
  doc?: string;
  signature?: string;
  receiver?: Receiver;
  typeParams?: string[];
  exported: boolean;
  children?: Symbol[];
  tags?: string[];
  language: 'ts';
}

export interface File {
  id: string;
  path: string;
  packageId: string;
  size: number;
  lineCount: number;
  buildTags?: string[];
  sha256: string;
  language: 'ts';
}

export interface Package {
  id: string;
  importPath: string;
  name: string;
  doc?: string;
  fileIds: string[];
  symbolIds: string[];
  language: 'ts';
}

export interface Ref {
  fromSymbolId: string;
  toSymbolId: string;
  kind: string;
  fileId: string;
  range: Range;
}

export interface Index {
  version: string;
  generatedAt: string;
  module: string;
  language: 'ts';
  packages: Package[];
  files: File[];
  symbols: Symbol[];
  refs: Ref[];
}
