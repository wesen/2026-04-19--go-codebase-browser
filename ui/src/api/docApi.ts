import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

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
  baseQuery: fetchBaseQuery({ baseUrl: '/api' }),
  keepUnusedDataFor: 3600,
  endpoints: (b) => ({
    listDocs: b.query<PageMeta[], void>({ query: () => '/doc' }),
    getDoc: b.query<DocPage, string>({ query: (slug) => `/doc/${encodeURIComponent(slug)}` }),
  }),
});

export const { useListDocsQuery, useGetDocQuery } = docApi;
