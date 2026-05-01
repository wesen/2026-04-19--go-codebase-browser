import { createApi } from '@reduxjs/toolkit/query/react';
import { fetchBaseQuery, type BaseQueryFn } from '@reduxjs/toolkit/query/react';
import { isStaticExport } from './runtimeMode';
import { getCommitDiff, getCommits, getImpact, getSymbolHistory } from './wasmClient';

export interface CommitRow {
  Hash: string;
  ShortHash: string;
  Message: string;
  AuthorName: string;
  AuthorEmail: string;
  AuthorTime: number;
  IndexedAt: number;
  Branch: string;
  Error: string;
}

export interface SymbolAtCommit {
  id: string;
  kind: string;
  name: string;
  packageId: string;
  fileId: string;
  startLine: number;
  endLine: number;
  signature: string;
  exported: boolean;
  bodyHash: string;
}

export interface SymbolHistoryEntry {
  commitHash: string;
  shortHash: string;
  message: string;
  authorTime: number;
  bodyHash: string;
  startLine: number;
  endLine: number;
  signature: string;
  kind: string;
}

export interface FileDiff {
  FileID: string;
  Path: string;
  ChangeType: string;
  OldSHA256: string;
  NewSHA256: string;
}

export interface SymbolDiff {
  SymbolID: string;
  Name: string;
  Kind: string;
  PackageID: string;
  ChangeType: string;
  OldStartLine: number;
  OldEndLine: number;
  NewStartLine: number;
  NewEndLine: number;
  OldSignature: string;
  NewSignature: string;
  OldBodyHash: string;
  NewBodyHash: string;
}

export interface DiffStats {
  FilesAdded: number;
  FilesRemoved: number;
  FilesModified: number;
  SymbolsAdded: number;
  SymbolsRemoved: number;
  SymbolsModified: number;
  SymbolsMoved: number;
  SymbolsUnchanged: number;
}

export interface CommitDiff {
  OldHash: string;
  NewHash: string;
  Files: FileDiff[];
  Symbols: SymbolDiff[];
  Stats: DiffStats;
}

export interface BodyDiffResult {
  symbolId: string;
  name: string;
  oldCommit: string;
  newCommit: string;
  oldBody: string;
  newBody: string;
  unifiedDiff: string;
  oldRange: string;
  newRange: string;
}

export interface ImpactEdge {
  fromSymbolId: string;
  toSymbolId: string;
  kind: string;
  fileId: string;
}

export interface ImpactNode {
  symbolId: string;
  name: string;
  kind: string;
  depth: number;
  edges: ImpactEdge[];
  compatibility: string;
  local: boolean;
}

export interface ImpactResponse {
  root: string;
  direction: string;
  depth: number;
  commit: string;
  nodes: ImpactNode[];
}

type StaticCommit = {
  hash: string;
  shortHash: string;
  message: string;
  authorName: string;
  authorTime: number;
};

type StaticBaseError = { status: string; data?: string };

const serverHistoryBaseQuery = fetchBaseQuery({ baseUrl: '/api/history' });

const historyBaseQuery: BaseQueryFn<string, unknown, StaticBaseError> = async (arg, api, extraOptions) => {
  if (!isStaticExport()) {
    return serverHistoryBaseQuery(arg, api, extraOptions) as any;
  }

  try {
    return await staticHistoryBaseQuery(arg);
  } catch (err) {
    return { error: { status: 'STATIC_HISTORY_ERROR', data: String(err) } };
  }
};

async function staticHistoryBaseQuery(arg: string): Promise<{ data?: unknown; error?: StaticBaseError }> {
  const commits = await loadStaticCommits();
  const commitByHash = new Map(commits.map((c) => [c.hash, c]));

  if (arg === '/commits') {
    return { data: commits.map(toCommitRow) };
  }

  const commitMatch = arg.match(/^\/commits\/([^/]+)$/);
  if (commitMatch) {
    const hash = resolveCommitRef(decodeURIComponent(commitMatch[1]), commits);
    const commit = hash ? commitByHash.get(hash) : undefined;
    return commit ? { data: toCommitRow(commit) } : staticNotFound(`commit not found: ${commitMatch[1]}`);
  }

  if (/^\/commits\/[^/]+\/symbols$/.test(arg)) {
    return { data: [] satisfies SymbolAtCommit[] };
  }

  if (arg.startsWith('/diff?')) {
    const params = paramsFor(arg);
    const from = resolveCommitRef(params.get('from') ?? '', commits);
    const to = resolveCommitRef(params.get('to') ?? '', commits);
    if (!from || !to) return staticNotFound(`cannot resolve diff refs: ${params.get('from')}..${params.get('to')}`);
    const diff = await getCommitDiff(from, to) as any;
    if (!diff) return staticNotFound(`diff not precomputed: ${from}..${to}`);
    return { data: normalizeDiff(diff) };
  }

  const historyMatch = arg.match(/^\/symbols\/(.+)\/history(?:\?(.*))?$/);
  if (historyMatch) {
    const symbolId = decodeURIComponent(historyMatch[1]);
    const query = new URLSearchParams(historyMatch[2] ?? '');
    const limit = Number.parseInt(query.get('limit') ?? '', 10);
    let entries = await getSymbolHistory(symbolId) as any[] | null;
    entries = (entries ?? []).map((entry) => normalizeHistoryEntry(entry, commitByHash));
    if (Number.isFinite(limit) && limit > 0) entries = entries.slice(0, limit);
    return { data: entries };
  }

  if (arg.startsWith('/symbol-body-diff?')) {
    // Body-level diffs are not in reviewData yet. Return a typed static error
    // instead of probing a nonexistent server API.
    return { error: { status: 'STATIC_NOT_PRECOMPUTED', data: 'symbol body diffs are not precomputed in static export yet' } };
  }

  if (arg.startsWith('/impact?')) {
    const params = paramsFor(arg);
    const sym = params.get('sym') ?? '';
    const dir = params.get('dir') ?? 'usedby';
    const depth = Number.parseInt(params.get('depth') ?? '2', 10);
    const impact = await getImpact(sym, dir, Number.isFinite(depth) ? depth : 2) as any;
    if (!impact) return staticNotFound(`impact not precomputed: ${sym}`);
    return { data: normalizeImpact(impact, commits) };
  }

  return { error: { status: 'UNKNOWN_STATIC_HISTORY_ENDPOINT', data: arg } };
}

async function loadStaticCommits(): Promise<StaticCommit[]> {
  const data = await getCommits() as StaticCommit[] | null;
  return data ?? [];
}

function resolveCommitRef(ref: string, commits: StaticCommit[]): string | null {
  if (!ref) return null;
  const ordered = [...commits].sort((a, b) => a.authorTime - b.authorTime);
  const newestIndex = ordered.length - 1;

  if (ref === 'HEAD') return ordered[newestIndex]?.hash ?? null;
  const headOffset = ref.match(/^HEAD~(\d+)$/);
  if (headOffset) {
    const index = newestIndex - Number.parseInt(headOffset[1], 10);
    return ordered[index]?.hash ?? null;
  }

  const exact = ordered.find((c) => c.hash === ref);
  if (exact) return exact.hash;
  const byShort = ordered.find((c) => c.shortHash === ref || c.hash.startsWith(ref));
  return byShort?.hash ?? null;
}

function paramsFor(path: string): URLSearchParams {
  const q = path.indexOf('?');
  return new URLSearchParams(q >= 0 ? path.slice(q + 1) : '');
}

function staticNotFound(message: string): { error: StaticBaseError } {
  return { error: { status: 'STATIC_NOT_FOUND', data: message } };
}

function toCommitRow(commit: StaticCommit): CommitRow {
  return {
    Hash: commit.hash,
    ShortHash: commit.shortHash,
    Message: commit.message,
    AuthorName: commit.authorName,
    AuthorEmail: '',
    AuthorTime: commit.authorTime,
    IndexedAt: 0,
    Branch: '',
    Error: '',
  };
}

function normalizeDiff(diff: any): CommitDiff {
  return {
    OldHash: diff.OldHash ?? diff.oldHash ?? '',
    NewHash: diff.NewHash ?? diff.newHash ?? '',
    Files: (diff.Files ?? diff.files ?? []).map(normalizeFileDiff),
    Symbols: (diff.Symbols ?? diff.symbols ?? []).map(normalizeSymbolDiff),
    Stats: normalizeStats(diff.Stats ?? diff.stats ?? {}),
  };
}

function normalizeStats(stats: any): DiffStats {
  return {
    FilesAdded: stats.FilesAdded ?? stats.filesAdded ?? 0,
    FilesRemoved: stats.FilesRemoved ?? stats.filesRemoved ?? 0,
    FilesModified: stats.FilesModified ?? stats.filesModified ?? 0,
    SymbolsAdded: stats.SymbolsAdded ?? stats.symbolsAdded ?? 0,
    SymbolsRemoved: stats.SymbolsRemoved ?? stats.symbolsRemoved ?? 0,
    SymbolsModified: stats.SymbolsModified ?? stats.symbolsModified ?? 0,
    SymbolsMoved: stats.SymbolsMoved ?? stats.symbolsMoved ?? 0,
    SymbolsUnchanged: stats.SymbolsUnchanged ?? stats.symbolsUnchanged ?? 0,
  };
}

function normalizeFileDiff(file: any): FileDiff {
  return {
    FileID: file.FileID ?? file.fileId ?? '',
    Path: file.Path ?? file.path ?? '',
    ChangeType: file.ChangeType ?? file.changeType ?? '',
    OldSHA256: file.OldSHA256 ?? file.oldSha256 ?? '',
    NewSHA256: file.NewSHA256 ?? file.newSha256 ?? '',
  };
}

function normalizeSymbolDiff(sym: any): SymbolDiff {
  return {
    SymbolID: sym.SymbolID ?? sym.symbolId ?? '',
    Name: sym.Name ?? sym.name ?? '',
    Kind: sym.Kind ?? sym.kind ?? '',
    PackageID: sym.PackageID ?? sym.packageId ?? '',
    ChangeType: sym.ChangeType ?? sym.changeType ?? '',
    OldStartLine: sym.OldStartLine ?? sym.oldStartLine ?? 0,
    OldEndLine: sym.OldEndLine ?? sym.oldEndLine ?? 0,
    NewStartLine: sym.NewStartLine ?? sym.newStartLine ?? 0,
    NewEndLine: sym.NewEndLine ?? sym.newEndLine ?? 0,
    OldSignature: sym.OldSignature ?? sym.oldSignature ?? '',
    NewSignature: sym.NewSignature ?? sym.newSignature ?? '',
    OldBodyHash: sym.OldBodyHash ?? sym.oldBodyHash ?? '',
    NewBodyHash: sym.NewBodyHash ?? sym.newBodyHash ?? '',
  };
}

function normalizeHistoryEntry(entry: any, commitByHash: Map<string, StaticCommit>): SymbolHistoryEntry {
  const commitHash = entry.commitHash ?? '';
  const commit = commitByHash.get(commitHash);
  return {
    commitHash,
    shortHash: entry.shortHash ?? commit?.shortHash ?? commitHash.slice(0, 7),
    message: entry.message ?? commit?.message ?? '',
    authorTime: entry.authorTime ?? commit?.authorTime ?? 0,
    bodyHash: entry.bodyHash ?? '',
    startLine: entry.startLine ?? 0,
    endLine: entry.endLine ?? 0,
    signature: entry.signature ?? '',
    kind: entry.kind ?? '',
  };
}

function normalizeImpact(impact: any, commits: StaticCommit[]): ImpactResponse {
  const ordered = [...commits].sort((a, b) => a.authorTime - b.authorTime);
  const newest = ordered.length > 0 ? ordered[ordered.length - 1] : undefined;
  return {
    root: impact.root ?? impact.rootSymbol ?? '',
    direction: impact.direction ?? 'usedby',
    depth: impact.depth ?? 0,
    commit: impact.commit ?? newest?.hash ?? '',
    nodes: (impact.nodes ?? []).map((node: any) => ({
      symbolId: node.symbolId ?? '',
      name: node.name ?? node.symbolId ?? '',
      kind: node.kind ?? '',
      depth: node.depth ?? 0,
      edges: node.edges ?? [],
      compatibility: node.compatibility ?? 'unknown',
      local: node.local ?? true,
    })),
  };
}

export const historyApi = createApi({
  reducerPath: 'historyApi',
  baseQuery: historyBaseQuery,
  endpoints: (builder) => ({
    listCommits: builder.query<CommitRow[], void>({
      query: () => '/commits',
    }),
    getCommit: builder.query<CommitRow, string>({
      query: (hash) => `/commits/${hash}`,
    }),
    getCommitSymbols: builder.query<SymbolAtCommit[], string>({
      query: (hash) => `/commits/${hash}/symbols`,
    }),
    getDiff: builder.query<CommitDiff, { from: string; to: string }>({
      query: ({ from, to }) => `/diff?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    }),
    getSymbolHistory: builder.query<SymbolHistoryEntry[], { symbolId: string; limit?: number }>({
      query: ({ symbolId, limit }) =>
        `/symbols/${encodeURIComponent(symbolId)}/history${limit ? `?limit=${limit}` : ''}`,
    }),
    getSymbolBodyDiff: builder.query<BodyDiffResult, { from: string; to: string; symbolId: string }>(
      {
        query: ({ from, to, symbolId }) =>
          `/symbol-body-diff?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&symbol=${encodeURIComponent(symbolId)}`,
      },
    ),
    getImpact: builder.query<ImpactResponse, { sym: string; dir?: 'usedby' | 'uses'; depth?: number; commit?: string }>({
      query: ({ sym, dir = 'usedby', depth = 2, commit }) => {
        const params = new URLSearchParams({ sym, dir, depth: String(depth) });
        if (commit) params.set('commit', commit);
        return `/impact?${params.toString()}`;
      },
    }),
  }),
});

export const {
  useListCommitsQuery,
  useGetCommitQuery,
  useGetCommitSymbolsQuery,
  useGetDiffQuery,
  useGetSymbolHistoryQuery,
  useGetSymbolBodyDiffQuery,
  useGetImpactQuery,
} = historyApi;
