import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

export type SnippetKind = 'declaration' | 'body' | 'signature';

export interface SnippetRefView {
  toSymbolId: string;
  kind: string;
  offsetInSnippet: number;
  length: number;
}

export interface SourceRefView {
  toSymbolId: string;
  kind: string;
  offset: number;
  length: number;
}

export interface FileXrefRef {
  fromSymbolId: string;
  toSymbolId: string;
  kind: string;
  fileId: string;
  range: { startLine: number; startCol: number; endLine: number; endCol: number; startOffset: number; endOffset: number };
}

export interface FileXrefUseTarget {
  toSymbolId: string;
  kind: string;
  count: number;
  occurrences: FileXrefRef[];
}

export interface FileXrefResponse {
  path: string;
  usedBy: FileXrefRef[];
  uses: FileXrefUseTarget[];
}

export const sourceApi = createApi({
  reducerPath: 'sourceApi',
  // Default query returns text; endpoints that need JSON override responseHandler.
  baseQuery: fetchBaseQuery({ baseUrl: '/api', responseHandler: 'text' }),
  keepUnusedDataFor: 3600,
  endpoints: (b) => ({
    getSource: b.query<string, string>({
      query: (path) => `/source?path=${encodeURIComponent(path)}`,
    }),
    getSnippet: b.query<string, { sym: string; kind?: SnippetKind }>({
      query: ({ sym, kind = 'declaration' }) =>
        `/snippet?sym=${encodeURIComponent(sym)}&kind=${kind}`,
    }),
    getSnippetRefs: b.query<SnippetRefView[], string>({
      query: (sym) => ({
        url: `/snippet-refs?sym=${encodeURIComponent(sym)}`,
        responseHandler: 'json',
      }),
    }),
    getSourceRefs: b.query<SourceRefView[], string>({
      query: (path) => ({
        url: `/source-refs?path=${encodeURIComponent(path)}`,
        responseHandler: 'json',
      }),
    }),
    getFileXref: b.query<FileXrefResponse, string>({
      query: (path) => ({
        url: `/file-xref?path=${encodeURIComponent(path)}`,
        responseHandler: 'json',
      }),
    }),
  }),
});

export const {
  useGetSourceQuery,
  useGetSnippetQuery,
  useGetSnippetRefsQuery,
  useGetSourceRefsQuery,
  useGetFileXrefQuery,
} = sourceApi;
