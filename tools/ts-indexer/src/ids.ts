// ID helpers mirroring internal/indexer/id.go. Keep the schemes aligned so
// Merge() in the Go indexer never sees collisions across languages.

import type { Kind } from './types.js';

export function symbolID(importPath: string, kind: Kind, name: string): string {
  return `sym:${importPath}.${kind}.${name}`;
}

export function methodID(importPath: string, recv: string, name: string): string {
  // Strip leading '*' if a TS receiver ever carries it (rare in TS); match Go.
  const r = recv.startsWith('*') ? recv.slice(1) : recv;
  return `sym:${importPath}.method.${r}.${name}`;
}

export function packageID(importPath: string): string {
  return `pkg:${importPath}`;
}

export function fileID(relPath: string): string {
  return `file:${relPath}`;
}
