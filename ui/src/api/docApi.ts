import { createApi } from '@reduxjs/toolkit/query/react';
import { wasmBaseQuery } from './wasmClient';

export interface SnippetRef {
  stubId: string;
  directive: string;
  symbolId?: string;
  filePath?: string;
  kind?: string;
  language?: string;
  text: string;
  startLine?: number;
  endLine?: number;
}

export interface DocPage {
  slug: string;
  title: string;
  html: string;
  snippets: SnippetRef[];
  errors?: string[];
}

export interface PageMeta {
  slug: string;
  title: string;
  path: string;
}

export const docApi = createApi({
  reducerPath: 'docApi',
  baseQuery: wasmBaseQuery,
  keepUnusedDataFor: 3600,
  endpoints: (b) => ({
    listDocs: b.query<PageMeta[], void>({
      queryFn: async (_arg, api, extraOptions) => {
        // In server-backed mode, prefer the live /api/doc endpoint so newly
        // added markdown pages are visible without regenerating the static
        // WASM precomputed bundle. Fall back to WASM for static deployments.
        try {
          const resp = await fetch('/api/doc');
          if (resp.ok) return { data: await resp.json() };
        } catch {}
        return wasmBaseQuery('docPages', api as any, extraOptions as any) as any;
      },
    }),
    getDoc: b.query<DocPage, string>({
      queryFn: async (slug, api, extraOptions) => {
        try {
          const resp = await fetch(`/api/doc/${encodeURIComponent(slug)}`);
          if (resp.ok) return { data: await resp.json() };
        } catch {}
        return wasmBaseQuery(`docPage:${slug}`, api as any, extraOptions as any) as any;
      },
    }),
  }),
});

export const { useListDocsQuery, useGetDocQuery } = docApi;
