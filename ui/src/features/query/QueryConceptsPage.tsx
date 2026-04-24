import React from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  useExecuteQueryConceptMutation,
  useListQueryConceptsQuery,
  type ExecuteQueryConceptResponse,
  type QueryConcept,
  type QueryConceptParam,
} from '../../api/conceptsApi';

export function QueryConceptsPage() {
  const params = useParams();
  const selectedPath = params['*'] ?? '';
  const { data, isLoading, error } = useListQueryConceptsQuery();

  if (isLoading) {
    return <div data-part="loading">Loading structured query concepts…</div>;
  }
  if (error) {
    return (
      <div>
        <h1 style={{ marginTop: 0 }}>Structured query concepts</h1>
        <p data-part="error">
          This page needs the server-backed query API. Run the site through <code>codebase-browser serve</code>{' '}
          with the SQLite DB available.
        </p>
      </div>
    );
  }

  const concepts = data ?? [];
  const concept = concepts.find((item) => item.path === selectedPath) ?? null;
  const groups = groupConcepts(concepts);

  return (
    <div>
      <h1 style={{ marginTop: 0 }}>Structured query concepts</h1>
      <p style={{ color: 'var(--cb-color-muted)' }}>
        These are named SQL concepts backed by the SQLite codebase index. Pick one to inspect its
        parameters, preview the rendered SQL, execute it against <code>codebase.db</code>, and jump from
        the results back into packages, source files, and symbols.
      </p>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'minmax(260px, 340px) 1fr',
          gap: 24,
          alignItems: 'start',
        }}
      >
        <aside
          style={{
            border: '1px solid var(--cb-color-border)',
            borderRadius: 12,
            padding: 16,
            background: 'var(--cb-color-surface, transparent)',
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 12 }}>
            <div style={{ fontWeight: 700 }}>Available concepts</div>
            <div style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>{concepts.length} total</div>
          </div>
          {concepts.length === 0 ? (
            <div data-part="empty">No concepts available.</div>
          ) : (
            <div style={{ display: 'grid', gap: 14 }}>
              {groups.map((group) => (
                <section key={group.folder || 'root'}>
                  <div style={{ fontSize: 12, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', color: 'var(--cb-color-muted)', marginBottom: 8 }}>
                    {group.folder || 'root'}
                  </div>
                  <ul data-part="tree-nav" style={{ margin: 0, paddingLeft: 18 }}>
                    {group.items.map((item) => {
                      const active = item.path === selectedPath;
                      return (
                        <li key={item.path} style={{ marginBottom: 8 }}>
                          <Link
                            to={`/queries/${item.path}`}
                            data-part="tree-node"
                            style={{
                              fontWeight: active ? 700 : 500,
                              color: active ? 'var(--cb-color-accent)' : undefined,
                              textDecoration: 'none',
                            }}
                          >
                            {item.name}
                          </Link>
                          <div data-role="hint" style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
                            {item.short}
                          </div>
                        </li>
                      );
                    })}
                  </ul>
                </section>
              ))}
            </div>
          )}
        </aside>
        <section>
          {concept ? (
            <QueryConceptDetail concept={concept} />
          ) : (
            <div
              data-part="empty"
              style={{ border: '1px dashed var(--cb-color-border)', borderRadius: 12, padding: 24 }}
            >
              Pick a concept from the left to inspect and run it.
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

function QueryConceptDetail({ concept }: { concept: QueryConcept }) {
  const [values, setValues] = React.useState<Record<string, unknown>>(() => initialValues(concept));
  const [result, setResult] = React.useState<ExecuteQueryConceptResponse | null>(null);
  const [executeQueryConcept, executeState] = useExecuteQueryConceptMutation();

  React.useEffect(() => {
    setValues(initialValues(concept));
    setResult(null);
  }, [concept]);

  async function run(renderOnly: boolean) {
    const params = buildExecutionParams(concept.params, values);
    const response = await executeQueryConcept({ path: concept.path, params, renderOnly }).unwrap();
    setResult(response);
  }

  return (
    <div style={{ display: 'grid', gap: 18 }}>
      <section style={{ border: '1px solid var(--cb-color-border)', borderRadius: 12, padding: 18 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, alignItems: 'start', flexWrap: 'wrap' }}>
          <div>
            <div style={{ fontSize: 12, color: 'var(--cb-color-muted)', marginBottom: 4 }}>{concept.folder || 'root'}</div>
            <h2 style={{ margin: 0 }}>{concept.path}</h2>
            <p style={{ marginBottom: 0 }}>{concept.long || concept.short}</p>
          </div>
          <div style={{ fontSize: 12, color: 'var(--cb-color-muted)', textAlign: 'right' }}>
            {concept.sourceRoot && <div>Source root: {concept.sourceRoot}</div>}
            {concept.sourcePath && <div>Source path: {concept.sourcePath}</div>}
          </div>
        </div>

        {!!concept.tags?.length && (
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 14 }}>
            {concept.tags.map((tag) => (
              <span
                key={tag}
                style={{ border: '1px solid var(--cb-color-border)', borderRadius: 999, padding: '2px 8px', fontSize: 12 }}
              >
                {tag}
              </span>
            ))}
          </div>
        )}
      </section>

      <section style={{ border: '1px solid var(--cb-color-border)', borderRadius: 12, padding: 18 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 14, gap: 12, flexWrap: 'wrap' }}>
          <h3 style={{ margin: 0 }}>Parameters</h3>
          <div style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
            {concept.params.length === 0 ? 'No parameters' : `${concept.params.length} parameter(s)`}
          </div>
        </div>

        <form
          onSubmit={(event) => {
            event.preventDefault();
            void run(false);
          }}
          style={{ display: 'grid', gap: 16 }}
        >
          {concept.params.length === 0 ? (
            <div data-part="symbol-doc">This concept has no parameters.</div>
          ) : (
            concept.params.map((param) => (
              <QueryParamField
                key={param.name}
                param={param}
                value={values[param.name]}
                onChange={(value) => setValues((prev) => ({ ...prev, [param.name]: value }))}
              />
            ))
          )}

          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'center' }}>
            <button type="submit" disabled={executeState.isLoading}>
              {executeState.isLoading ? 'Running…' : 'Run query'}
            </button>
            <button type="button" disabled={executeState.isLoading} onClick={() => void run(true)}>
              Render SQL only
            </button>
            {result && (
              <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
                {result.rowCount} row(s)
                {result.rows && result.rows.length > 0 ? ' returned' : result.renderedSql ? ' rendered' : ''}
              </span>
            )}
          </div>
        </form>
      </section>

      {executeState.error && (
        <pre
          data-part="error"
          style={{ whiteSpace: 'pre-wrap', border: '1px solid var(--cb-color-border)', borderRadius: 12, padding: 16 }}
        >
          {JSON.stringify(executeState.error, null, 2)}
        </pre>
      )}

      {result && (
        <>
          <section style={{ border: '1px solid var(--cb-color-border)', borderRadius: 12, padding: 18 }}>
            <h3 style={{ marginTop: 0, marginBottom: 8 }}>Rendered SQL</h3>
            <pre
              style={{
                whiteSpace: 'pre-wrap',
                overflowX: 'auto',
                padding: 12,
                border: '1px solid var(--cb-color-border)',
                borderRadius: 8,
                margin: 0,
              }}
            >
              {result.renderedSql}
            </pre>
          </section>

          <section style={{ border: '1px solid var(--cb-color-border)', borderRadius: 12, padding: 18 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
              <h3 style={{ margin: 0 }}>Results</h3>
              <div style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
                {result.rowCount} row(s)
                {result.rows && result.rows.length > 0 ? ' • linked where possible' : ''}
              </div>
            </div>
            {result.rows && result.rows.length > 0 && result.columns && result.columns.length > 0 ? (
              <ResultTable columns={result.columns} rows={result.rows} />
            ) : (
              <div data-part="symbol-doc">
                {result.rowCount > 0 ? `${result.rowCount} row(s)` : 'No rows returned or render-only mode was used.'}
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function ResultTable({ columns, rows }: { columns: string[]; rows: Record<string, unknown>[] }) {
  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={{ borderCollapse: 'collapse', width: '100%' }}>
        <thead>
          <tr>
            {columns.map((column) => (
              <th
                key={column}
                style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}
              >
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr key={index}>
              {columns.map((column) => (
                <td
                  key={column}
                  style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px', verticalAlign: 'top' }}
                >
                  <ResultCell column={column} value={row[column]} row={row} />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function QueryParamField({ param, value, onChange }: { param: QueryConceptParam; value: unknown; onChange: (value: unknown) => void }) {
  return (
    <label style={{ display: 'grid', gap: 6 }}>
      <span style={{ fontWeight: 600 }}>
        {param.name}
        {param.required ? ' *' : ''}
      </span>
      <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>{param.help || param.type}</span>
      {renderField(param, value, onChange)}
    </label>
  );
}

function renderField(param: QueryConceptParam, value: unknown, onChange: (value: unknown) => void) {
  switch (param.type) {
    case 'bool':
      return <input type="checkbox" checked={Boolean(value)} onChange={(event) => onChange(event.target.checked)} />;
    case 'choice':
      return (
        <select value={String(value ?? '')} onChange={(event) => onChange(event.target.value)}>
          <option value="">—</option>
          {(param.choices ?? []).map((choice) => (
            <option key={choice} value={choice}>
              {choice}
            </option>
          ))}
        </select>
      );
    case 'stringList':
    case 'intList':
      return (
        <input
          type="text"
          value={String(value ?? '')}
          onChange={(event) => onChange(event.target.value)}
          placeholder="comma,separated,values"
        />
      );
    case 'int':
      return <input type="number" value={String(value ?? '')} onChange={(event) => onChange(event.target.value)} />;
    default:
      return <input type="text" value={String(value ?? '')} onChange={(event) => onChange(event.target.value)} />;
  }
}

function ResultCell({ column, value, row }: { column: string; value: unknown; row: Record<string, unknown> }) {
  const text = formatCell(value);
  const target = inferCellTarget(column, value, row);

  if (!target || text === '') {
    return <code>{text}</code>;
  }

  return (
    <Link to={target.href} style={{ textDecoration: 'none' }}>
      <code>{text}</code>
    </Link>
  );
}

function inferCellTarget(column: string, value: unknown, row: Record<string, unknown>): { href: string } | null {
  if (typeof value !== 'string' || value.trim() === '') {
    return null;
  }

  const text = value.trim();
  const lowerColumn = column.toLowerCase();

  if (text.startsWith('sym:')) {
    return { href: `/symbol/${encodeURIComponent(text)}` };
  }
  if (text.startsWith('pkg:')) {
    return { href: `/packages/${encodeURIComponent(text)}` };
  }
  if (text.startsWith('file:')) {
    return { href: `/source/${text.slice(5)}` };
  }

  if (lowerColumn === 'package' || lowerColumn === 'import_path' || lowerColumn.endsWith('package_id')) {
    const packageID = text.startsWith('pkg:') ? text : `pkg:${text}`;
    return { href: `/packages/${encodeURIComponent(packageID)}` };
  }

  if (lowerColumn === 'file' || lowerColumn === 'path' || lowerColumn.endsWith('file_id')) {
    const sourcePath = text.startsWith('file:') ? text.slice(5) : text;
    if (looksLikeSourcePath(sourcePath)) {
      return { href: `/source/${sourcePath}` };
    }
  }

  if ((lowerColumn === 'name' || lowerColumn.endsWith('_name')) && typeof row.id === 'string' && row.id.startsWith('sym:')) {
    return { href: `/symbol/${encodeURIComponent(row.id)}` };
  }
  if ((lowerColumn === 'name' || lowerColumn.endsWith('_name')) && typeof row.symbol_id === 'string' && row.symbol_id.startsWith('sym:')) {
    return { href: `/symbol/${encodeURIComponent(String(row.symbol_id))}` };
  }

  return null;
}

function looksLikeSourcePath(value: string): boolean {
  return /\.(go|ts|tsx|js|jsx|css|json|md|sql|yaml|yml)$/.test(value) || value.includes('/');
}

function groupConcepts(concepts: QueryConcept[]): Array<{ folder: string; items: QueryConcept[] }> {
  const byFolder = new Map<string, QueryConcept[]>();
  for (const concept of concepts) {
    const folder = concept.folder || '';
    const bucket = byFolder.get(folder) ?? [];
    bucket.push(concept);
    byFolder.set(folder, bucket);
  }
  return [...byFolder.entries()]
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([folder, items]) => ({
      folder,
      items: [...items].sort((a, b) => a.name.localeCompare(b.name)),
    }));
}

function initialValues(concept: QueryConcept): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  for (const param of concept.params) {
    if (param.type === 'bool') {
      values[param.name] = Boolean(param.default);
      continue;
    }
    if (Array.isArray(param.default)) {
      values[param.name] = param.default.join(', ');
      continue;
    }
    values[param.name] = param.default ?? '';
  }
  return values;
}

function buildExecutionParams(params: QueryConceptParam[], values: Record<string, unknown>): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const param of params) {
    const value = values[param.name];
    switch (param.type) {
      case 'bool':
        out[param.name] = Boolean(value);
        break;
      default:
        out[param.name] = value ?? '';
        break;
    }
  }
  return out;
}

function formatCell(value: unknown): string {
  if (value === null || value === undefined) {
    return '';
  }
  if (typeof value === 'object') {
    return JSON.stringify(value);
  }
  return String(value);
}
