import { createApi } from '@reduxjs/toolkit/query/react';
import type { Range } from './types';
import { wasmBaseQuery } from './wasmClient';

export interface RefRecord {
  fromSymbolId: string;
  toSymbolId: string;
  kind: string;
  fileId: string;
  range: Range;
}

export interface XrefUseTarget {
  toSymbolId: string;
  kind: string;
  count: number;
  occurrences: RefRecord[];
}

export interface XrefResponse {
  id: string;
  usedBy: RefRecord[];
  uses: XrefUseTarget[];
}

export const xrefApi = createApi({
  reducerPath: 'xrefApi',
  baseQuery: wasmBaseQuery,
  keepUnusedDataFor: 3600,
  endpoints: (b) => ({
    getXref: b.query<XrefResponse, string>({
      query: (id) => `xref:${id}`,
    }),
  }),
});

export const { useGetXrefQuery } = xrefApi;
