import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { IndexSummary, PackageLite, Symbol } from './types';

export const indexApi = createApi({
  reducerPath: 'indexApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api' }),
  tagTypes: ['Index', 'Package', 'Symbol'],
  // Binary is immutable; aggressive cache is fine.
  keepUnusedDataFor: 3600,
  endpoints: (b) => ({
    getIndex: b.query<IndexSummary, void>({
      query: () => '/index',
      providesTags: ['Index'],
    }),
    getPackages: b.query<PackageLite[], void>({
      query: () => '/packages',
      providesTags: ['Package'],
    }),
    getSymbol: b.query<Symbol, string>({
      query: (id) => `/symbol/${encodeURIComponent(id)}`,
      providesTags: (_r, _e, id) => [{ type: 'Symbol', id }],
    }),
    searchSymbols: b.query<Symbol[], { q: string; kind?: string }>({
      query: ({ q, kind }) => `/search?q=${encodeURIComponent(q)}&kind=${kind ?? ''}`,
    }),
  }),
});

export const {
  useGetIndexQuery,
  useGetPackagesQuery,
  useGetSymbolQuery,
  useSearchSymbolsQuery,
} = indexApi;
