import React from 'react';
import { Link } from 'react-router-dom';
import {
  useListCommitsQuery,
  useGetDiffQuery,
  type CommitRow,
  type SymbolDiff,
  type FileDiff,
} from '../../api/historyApi';

export function HistoryPage() {
  const { data: commits, isLoading, error } = useListCommitsQuery();

  if (isLoading) {
    return <div data-part="loading">Loading commit history…</div>;
  }
  if (error) {
    return (
      <div>
        <h1 style={{ marginTop: 0 }}>Codebase history</h1>
        <p data-part="error">
          This page needs the server-backed history API. Run{' '}
          <code>codebase-browser serve --history-db history.db</code> after scanning commits.
        </p>
      </div>
    );
  }

  const rows = commits ?? [];
  return (
    <div>
      <h1 style={{ marginTop: 0 }}>Codebase history</h1>
      <p style={{ color: 'var(--cb-color-muted)' }}>
        {rows.length} indexed commit(s). Select two commits to diff.
      </p>
      {rows.length === 0 ? (
        <div data-part="empty">No commits indexed yet.</div>
      ) : (
        <CommitTimeline commits={rows} />
      )}
    </div>
  );
}

function CommitTimeline({ commits }: { commits: CommitRow[] }) {
  const [selectedOld, setSelectedOld] = React.useState('');
  const [selectedNew, setSelectedNew] = React.useState('');

  // Auto-select HEAD and HEAD~3 if available.
  React.useEffect(() => {
    if (commits.length >= 2 && !selectedOld && !selectedNew) {
      setSelectedNew(commits[0].Hash);
      if (commits.length > 3) {
        setSelectedOld(commits[3].Hash);
      } else {
        setSelectedOld(commits[commits.length - 1].Hash);
      }
    }
  }, [commits, selectedOld, selectedNew]);

  const diffQuery = useGetDiffQuery(
    { from: selectedOld, to: selectedNew },
    { skip: !selectedOld || !selectedNew || selectedOld === selectedNew },
  );

  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'minmax(260px, 380px) 1fr', gap: 24, alignItems: 'start' }}>
      <aside
        style={{
          border: '1px solid var(--cb-color-border)',
          borderRadius: 12,
          padding: 16,
          maxHeight: '70vh',
          overflowY: 'auto',
        }}
      >
        <div style={{ fontWeight: 700, marginBottom: 12 }}>Commits</div>
        <ul style={{ margin: 0, paddingLeft: 18, display: 'grid', gap: 6 }}>
          {commits.map((c) => {
            const isOld = c.Hash === selectedOld;
            const isNew = c.Hash === selectedNew;
            return (
              <li key={c.Hash} style={{ display: 'flex', gap: 6, alignItems: 'baseline' }}>
                <button
                  onClick={() => setSelectedOld(c.Hash)}
                  style={{
                    padding: '1px 6px',
                    fontSize: 11,
                    border: isOld ? '2px solid var(--cb-color-accent)' : '1px solid var(--cb-color-border)',
                    borderRadius: 4,
                    background: isOld ? 'var(--cb-color-accent)' : 'transparent',
                    color: isOld ? '#fff' : 'inherit',
                    cursor: 'pointer',
                  }}
                >
                  old
                </button>
                <button
                  onClick={() => setSelectedNew(c.Hash)}
                  style={{
                    padding: '1px 6px',
                    fontSize: 11,
                    border: isNew ? '2px solid #4caf50' : '1px solid var(--cb-color-border)',
                    borderRadius: 4,
                    background: isNew ? '#4caf50' : 'transparent',
                    color: isNew ? '#fff' : 'inherit',
                    cursor: 'pointer',
                  }}
                >
                  new
                </button>
                <span style={{ fontSize: 12 }}>
                  <code>{c.ShortHash}</code>{' '}
                  <span style={{ color: 'var(--cb-color-muted)' }}>{c.Message}</span>
                </span>
              </li>
            );
          })}
        </ul>
      </aside>

      <section>
        {selectedOld && selectedNew && selectedOld !== selectedNew ? (
          diffQuery.isLoading ? (
            <div>Loading diff…</div>
          ) : diffQuery.error ? (
            <pre data-part="error" style={{ whiteSpace: 'pre-wrap' }}>
              {JSON.stringify(diffQuery.error, null, 2)}
            </pre>
          ) : diffQuery.data ? (
            <DiffView diff={diffQuery.data} />
          ) : null
        ) : (
          <div
            style={{ border: '1px dashed var(--cb-color-border)', borderRadius: 12, padding: 24 }}
          >
            Select an <strong>old</strong> and a <strong>new</strong> commit from the list to see the diff.
          </div>
        )}
      </section>
    </div>
  );
}

function DiffView({ diff }: { diff: ReturnType<typeof useGetDiffQuery>['data'] & {} }) {
  if (!diff) return null;

  return (
    <div style={{ display: 'grid', gap: 18 }}>
      <section style={{ border: '1px solid var(--cb-color-border)', borderRadius: 12, padding: 16 }}>
        <h3 style={{ marginTop: 0 }}>
          Diff: <code>{diff.OldHash.slice(0, 7)}</code> → <code>{diff.NewHash.slice(0, 7)}</code>
        </h3>
        <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', fontSize: 14 }}>
          <span>Files: +{diff.Stats.FilesAdded} -{diff.Stats.FilesRemoved} ~{diff.Stats.FilesModified}</span>
          <span>
            Symbols: +{diff.Stats.SymbolsAdded} -{diff.Stats.SymbolsRemoved} ~{diff.Stats.SymbolsModified} →{diff.Stats.SymbolsMoved}
          </span>
        </div>
      </section>

      {diff.Files && diff.Files.length > 0 && (
        <section style={{ border: '1px solid var(--cb-color-border)', borderRadius: 12, padding: 16 }}>
          <h3 style={{ marginTop: 0 }}>Changed files</h3>
          <ul style={{ margin: 0, paddingLeft: 18 }}>
            {diff.Files.map((f: FileDiff) => (
              <li key={f.Path} style={{ marginBottom: 4 }}>
                <code
                  style={{
                    color:
                      f.ChangeType === 'added'
                        ? '#4caf50'
                        : f.ChangeType === 'removed'
                          ? '#f44336'
                          : undefined,
                  }}
                >
                  {f.ChangeType}
                </code>{' '}
                <Link
                  to={`/source/${f.Path}`}
                  style={{ textDecoration: 'none' }}
                >
                  <code>{f.Path}</code>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}

      {diff.Symbols && diff.Symbols.length > 0 && (
        <section style={{ border: '1px solid var(--cb-color-border)', borderRadius: 12, padding: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
            <h3 style={{ margin: 0 }}>Changed symbols</h3>
            <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
              {diff.Symbols.length} changed
            </span>
          </div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ borderCollapse: 'collapse', width: '100%' }}>
              <thead>
                <tr>
                  <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Change</th>
                  <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Kind</th>
                  <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Name</th>
                  <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Lines</th>
                </tr>
              </thead>
              <tbody>
                {diff.Symbols.map((s: SymbolDiff) => (
                  <tr key={s.SymbolID}>
                    <td
                      style={{
                        borderBottom: '1px solid var(--cb-color-border)',
                        padding: '6px 8px',
                      }}
                    >
                      <code
                        style={{
                          fontSize: 11,
                          padding: '1px 4px',
                          borderRadius: 4,
                          border: '1px solid var(--cb-color-border)',
                          color:
                            s.ChangeType === 'added'
                              ? '#4caf50'
                              : s.ChangeType === 'removed'
                                ? '#f44336'
                                : s.ChangeType === 'modified'
                                  ? '#ff9800'
                                  : undefined,
                        }}
                      >
                        {s.ChangeType}
                      </code>
                    </td>
                    <td style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>
                      <code>{s.Kind}</code>
                    </td>
                    <td style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>
                      <Link
                        to={`/symbol/${encodeURIComponent(s.SymbolID)}`}
                        style={{ textDecoration: 'none' }}
                      >
                        <code>{s.Name}</code>
                      </Link>
                    </td>
                    <td style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px', fontSize: 12, color: 'var(--cb-color-muted)' }}>
                      {s.ChangeType === 'added'
                        ? `${s.NewStartLine}-${s.NewEndLine}`
                        : s.ChangeType === 'removed'
                          ? `${s.OldStartLine}-${s.OldEndLine}`
                          : `${s.OldStartLine}-${s.OldEndLine} → ${s.NewStartLine}-${s.NewEndLine}`}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {diff.Symbols && diff.Symbols.length === 0 && (
        <div style={{ border: '1px dashed var(--cb-color-border)', borderRadius: 12, padding: 16 }}>
          No symbol changes between these commits.
        </div>
      )}
    </div>
  );
}
