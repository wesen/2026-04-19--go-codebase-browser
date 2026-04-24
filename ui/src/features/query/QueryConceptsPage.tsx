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
          This page needs the server-backed query API. Run the site through <code>codebase-browser serve</code>
          {' '}with the SQLite DB available.
        </p>
      </div>
    );
  }
  const concepts = data ?? [];
  const concept = concepts.find((item) => item.path === selectedPath) ?? null;

  return (
    <div>
      <h1 style={{ marginTop: 0 }}>Structured query concepts</h1>
      <p style={{ color: 'var(--cb-color-muted)' }}>
        These are the named SQL concepts backed by the SQLite codebase index. Pick one to inspect its
        parameters, preview the rendered SQL, or execute it against <code>codebase.db</code>.
      </p>
      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(240px, 320px) 1fr', gap: 24, alignItems: 'start' }}>
        <aside>
          <div style={{ fontWeight: 600, marginBottom: 8 }}>Available concepts</div>
          {concepts.length === 0 ? (
            <div data-part="empty">No concepts available.</div>
          ) : (
            <ul data-part="tree-nav" style={{ margin: 0, paddingLeft: 18 }}>
              {concepts.map((item) => (
                <li key={item.path}>
                  <Link
                    to={`/queries/${item.path}`}
                    data-part="tree-node"
                    style={{ fontWeight: item.path === selectedPath ? 700 : 400 }}
                  >
                    {item.path}
                  </Link>
                  <div data-role="hint" style={{ fontSize: 12, color: 'var(--cb-color-muted)', marginBottom: 8 }}>
                    {item.short}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </aside>
        <section>
          {concept ? (
            <QueryConceptDetail concept={concept} />
          ) : (
            <div data-part="empty">Pick a concept from the list to inspect and run it.</div>
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
    <div>
      <h2 style={{ marginTop: 0 }}>{concept.path}</h2>
      <p style={{ marginTop: 0 }}>{concept.long || concept.short}</p>
      {!!concept.tags?.length && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}>
          {concept.tags.map((tag) => (
            <span key={tag} style={{ border: '1px solid var(--cb-color-border)', borderRadius: 999, padding: '2px 8px', fontSize: 12 }}>
              {tag}
            </span>
          ))}
        </div>
      )}

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
            <QueryParamField key={param.name} param={param} value={values[param.name]} onChange={(value) => setValues((prev) => ({ ...prev, [param.name]: value }))} />
          ))
        )}

        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          <button type="submit" disabled={executeState.isLoading}>
            Run query
          </button>
          <button type="button" disabled={executeState.isLoading} onClick={() => void run(true)}>
            Render SQL only
          </button>
        </div>
      </form>

      {executeState.error && (
        <pre data-part="error" style={{ whiteSpace: 'pre-wrap' }}>
          {JSON.stringify(executeState.error, null, 2)}
        </pre>
      )}

      {result && (
        <div style={{ marginTop: 24, display: 'grid', gap: 16 }}>
          <section>
            <h3 style={{ marginBottom: 8 }}>Rendered SQL</h3>
            <pre style={{ whiteSpace: 'pre-wrap', overflowX: 'auto', padding: 12, border: '1px solid var(--cb-color-border)', borderRadius: 8 }}>
              {result.renderedSql}
            </pre>
          </section>

          <section>
            <h3 style={{ marginBottom: 8 }}>Results</h3>
            {result.rows && result.rows.length > 0 && result.columns && result.columns.length > 0 ? (
              <div style={{ overflowX: 'auto' }}>
                <table style={{ borderCollapse: 'collapse', width: '100%' }}>
                  <thead>
                    <tr>
                      {result.columns.map((column) => (
                        <th key={column} style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>
                          {column}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {result.rows.map((row, index) => (
                      <tr key={index}>
                        {result.columns!.map((column) => (
                          <td key={column} style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px', verticalAlign: 'top' }}>
                            <code>{formatCell(row[column])}</code>
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div data-part="symbol-doc">
                {result.rowCount > 0 ? `${result.rowCount} row(s)` : 'No rows returned or render-only mode was used.'}
              </div>
            )}
          </section>
        </div>
      )}
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
      return (
        <input
          type="checkbox"
          checked={Boolean(value)}
          onChange={(event) => onChange(event.target.checked)}
        />
      );
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
      return (
        <input
          type="number"
          value={String(value ?? '')}
          onChange={(event) => onChange(event.target.value)}
        />
      );
    default:
      return (
        <input
          type="text"
          value={String(value ?? '')}
          onChange={(event) => onChange(event.target.value)}
        />
      );
  }
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
