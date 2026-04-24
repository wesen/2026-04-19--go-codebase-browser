import { createApi } from '@reduxjs/toolkit/query/react';
import { wasmBaseQuery } from './wasmClient';
import type { IndexSummary, PackageLite, Symbol } from './types';

export const indexApi = createApi({
  reducerPath: 'indexApi',
  baseQuery: wasmBaseQuery,
  tagTypes: ['Index', 'Package', 'Symbol'],
  // Binary is immutable; aggressive cache is fine.
  keepUnusedDataFor: 3600,
  endpoints: (b) => ({
    getIndex: b.query<IndexSummary, void>({
      query: () => 'index',
      providesTags: ['Index'],
    }),
    getPackages: b.query<PackageLite[], void>({
      query: () => 'packages',
      providesTags: ['Package'],
    }),
    getSymbol: b.query<Symbol, string>({
      query: (id) => `symbol:${id}`,
      providesTags: (_r, _e, id) => [{ type: 'Symbol', id }],
    }),
    searchSymbols: b.query<Symbol[], { q: string; kind?: string }>({
      query: ({ q, kind }) => `search:${q}|${kind ?? ''}`,
    }),
  }),
});

export const {
  useGetIndexQuery,
  useGetPackagesQuery,
  useGetSymbolQuery,
  useSearchSymbolsQuery,
} = indexApi;
