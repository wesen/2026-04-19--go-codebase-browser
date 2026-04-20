import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

export type SnippetKind = 'declaration' | 'body' | 'signature';

export const sourceApi = createApi({
  reducerPath: 'sourceApi',
  baseQuery: fetchBaseQuery({
    baseUrl: '/api',
    responseHandler: 'text',
  }),
  keepUnusedDataFor: 3600,
  endpoints: (b) => ({
    getSource: b.query<string, string>({
      query: (path) => `/source?path=${encodeURIComponent(path)}`,
    }),
    getSnippet: b.query<string, { sym: string; kind?: SnippetKind }>({
      query: ({ sym, kind = 'declaration' }) =>
        `/snippet?sym=${encodeURIComponent(sym)}&kind=${kind}`,
    }),
  }),
});

export const { useGetSourceQuery, useGetSnippetQuery } = sourceApi;
