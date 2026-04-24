import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

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

function encodeConceptPath(path: string): string {
  return path
    .split('/')
    .filter(Boolean)
    .map((segment) => encodeURIComponent(segment))
    .join('/');
}

export const conceptsApi = createApi({
  reducerPath: 'conceptsApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/' }),
  tagTypes: ['QueryConcept'],
  endpoints: (builder) => ({
    listQueryConcepts: builder.query<QueryConcept[], void>({
      query: () => 'query-concepts',
      providesTags: ['QueryConcept'],
    }),
    getQueryConcept: builder.query<QueryConcept, string>({
      query: (path) => `query-concepts/${encodeConceptPath(path)}`,
      providesTags: (_result, _error, path) => [{ type: 'QueryConcept', id: path }],
    }),
    executeQueryConcept: builder.mutation<ExecuteQueryConceptResponse, ExecuteQueryConceptRequest>({
      query: ({ path, params, renderOnly }) => ({
        url: `query-concepts/${encodeConceptPath(path)}/execute`,
        method: 'POST',
        body: { params: params ?? {}, renderOnly: !!renderOnly },
      }),
    }),
  }),
});

export const {
  useListQueryConceptsQuery,
  useGetQueryConceptQuery,
  useExecuteQueryConceptMutation,
} = conceptsApi;
