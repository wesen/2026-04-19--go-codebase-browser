export type QueryErrorCode =
  | 'NOT_FOUND'
  | 'AMBIGUOUS_REF'
  | 'SQL_ERROR'
  | 'DB_LOAD_ERROR'
  | 'FEATURE_UNAVAILABLE';

export class QueryError extends Error {
  constructor(
    public code: QueryErrorCode,
    message: string,
    public details: Record<string, unknown> = {},
  ) {
    super(message);
    this.name = 'QueryError';
  }
}

export function normalizeQueryError(err: unknown): { status: string; data?: string } {
  if (err instanceof QueryError) {
    return { status: err.code, data: err.message };
  }
  if (err instanceof Error) {
    return { status: 'SQL_ERROR', data: err.message };
  }
  return { status: 'SQL_ERROR', data: String(err) };
}
