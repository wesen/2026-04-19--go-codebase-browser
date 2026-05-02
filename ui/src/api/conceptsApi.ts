import { createApi, type BaseQueryFn } from '@reduxjs/toolkit/query/react';

export interface QueryConceptParam {
  name: string;
  type: 'string' | 'int' | 'bool' | 'choice' | 'stringList' | 'intList';
  help?: string;
  required?: boolean;
  default?: unknown;
  choices?: string[];
  shortFlag?: string;
}

export interface QueryConcept {
  path: string;
  name: string;
  folder?: string;
  short: string;
  long?: string;
  tags?: string[];
  params: QueryConceptParam[];
  sourceRoot?: string;
  sourcePath?: string;
  query?: string;
}

export interface ExecuteQueryConceptRequest {
  path: string;
  params?: Record<string, unknown>;
  renderOnly?: boolean;
}

export interface ExecuteQueryConceptResponse {
  conceptPath: string;
  renderedSql: string;
  columns?: string[];
  rows?: Record<string, unknown>[];
  rowCount: number;
  rendered: boolean;
}

type StaticError = { status: string; data?: string };

const noopBaseQuery: BaseQueryFn<void, unknown, StaticError> = async () => ({ data: undefined });

export const conceptsApi = createApi({
  reducerPath: 'conceptsApi',
  baseQuery: noopBaseQuery,
  tagTypes: ['QueryConcept'],
  endpoints: (builder) => ({
    listQueryConcepts: builder.query<QueryConcept[], void>({
      queryFn: async () => ({ data: [] }),
      providesTags: ['QueryConcept'],
    }),
    getQueryConcept: builder.query<QueryConcept, string>({
      queryFn: async (path) => ({ error: { status: 'FEATURE_UNAVAILABLE', data: `query concept not packaged: ${path}` } }),
      providesTags: (_result, _error, path) => [{ type: 'QueryConcept', id: path }],
    }),
    executeQueryConcept: builder.mutation<ExecuteQueryConceptResponse, ExecuteQueryConceptRequest>({
      queryFn: async ({ path }) => ({ error: { status: 'FEATURE_UNAVAILABLE', data: `query concept execution is not available in the static-only runtime: ${path}` } }),
    }),
  }),
});

export const {
  useListQueryConceptsQuery,
  useGetQueryConceptQuery,
  useExecuteQueryConceptMutation,
} = conceptsApi;
