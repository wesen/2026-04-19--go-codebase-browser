import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { wasmBaseQuery, getPrecomputed } from './wasmClient';

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

// Source files are served as static assets
const staticBaseQuery = fetchBaseQuery({ baseUrl: '' });

// Snippets come from WASM (precomputed.json loaded into WASM memory)
const snippetBaseQuery: typeof wasmBaseQuery = async (arg) => {
  if (!window.codebaseBrowser) {
    return { error: { status: 'WASM_ERROR', data: 'WASM not initialized' } };
  }
  try {
    const [sym, kind] = (arg as string).slice(8).split('|');
    const result = window.codebaseBrowser.getSnippet(sym, kind || 'declaration');
    return { data: JSON.parse(result) };
  } catch (err) {
    return { error: { status: 'WASM_ERROR', data: String(err) } };
  }
};

export const sourceApi = createApi({
  reducerPath: 'sourceApi',
  baseQuery: staticBaseQuery,
  keepUnusedDataFor: 3600,
  endpoints: (b) => ({
    // Static source file serving
    getSource: b.query<string, string>({
      query: (path) => `/source/${path}`,
    }),

    // Snippet from WASM
    getSnippet: b.query<string, { sym: string; kind?: SnippetKind }>({
      queryFn: async ({ sym, kind = 'declaration' }) => {
        const result = await snippetBaseQuery(`snippet:${sym}|${kind}`, undefined as any, undefined as any);
        if ('error' in result) return result as any;
        const obj = result.data as { text: string };
        return { data: obj.text };
      },
    }),

    // Snippet refs from precomputed.json
    getSnippetRefs: b.query<SnippetRefView[], string>({
      queryFn: async (sym) => {
        const pc = await getPrecomputed();
        const refs = (pc.snippetRefs as Record<string, SnippetRefView[]> | undefined)?.[sym] ?? [];
        return { data: refs };
      },
    }),

    // Source refs from precomputed.json
    getSourceRefs: b.query<SourceRefView[], string>({
      queryFn: async (path) => {
        const pc = await getPrecomputed();
        const refs = (pc.sourceRefs as Record<string, SourceRefView[]> | undefined)?.[path] ?? [];
        return { data: refs };
      },
    }),

    // File xref from precomputed.json
    getFileXref: b.query<FileXrefResponse, string>({
      queryFn: async (path) => {
        const pc = await getPrecomputed();
        const data = (pc.fileXrefIndex as Record<string, FileXrefResponse> | undefined)?.[path];
        if (!data) {
          return { data: { path, usedBy: [], uses: [] } };
        }
        return { data };
      },
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
