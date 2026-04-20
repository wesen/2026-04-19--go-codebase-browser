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
  kind: string;
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
  language?: string;
}

export interface File {
  id: string;
  path: string;
  packageId: string;
  size: number;
  lineCount: number;
  buildTags?: string[];
  sha256: string;
  language?: string;
}

export interface Package {
  id: string;
  importPath: string;
  name: string;
  doc?: string;
  fileIds: string[];
  symbolIds: string[];
  language?: string;
}

export interface PackageLite {
  id: string;
  importPath: string;
  name: string;
  files: number;
  symbols: number;
}

export interface IndexSummary {
  version: string;
  generatedAt: string;
  module: string;
  goVersion: string;
  packages: Package[];
  files: File[];
  symbols: Symbol[];
}
