import { createApi } from '@reduxjs/toolkit/query/react';
import { fetchBaseQuery } from '@reduxjs/toolkit/query/react';

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

export const historyApi = createApi({
  reducerPath: 'historyApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/history' }),
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
  }),
});

export const {
  useListCommitsQuery,
  useGetCommitQuery,
  useGetCommitSymbolsQuery,
  useGetDiffQuery,
  useGetSymbolHistoryQuery,
} = historyApi;
