import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  useListCommitsQuery,
  useGetDiffQuery,
  useGetSymbolHistoryQuery,
  useGetSymbolBodyDiffQuery,
  type CommitRow,
  type SymbolDiff,
  type SymbolHistoryEntry,
  type FileDiff,
} from '../../api/historyApi';

export function HistoryPage() {
  const { data: commits, isLoading, error } = useListCommitsQuery();
  const location = useLocation();
  const initialSymbol = React.useMemo(() => {
    const params = new URLSearchParams(location.search);
    return params.get('symbol') || '';
  }, [location.search]);

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
        <CommitTimeline commits={rows} initialSymbol={initialSymbol} />
      )}
    </div>
  );
}

function CommitTimeline({ commits, initialSymbol }: { commits: CommitRow[]; initialSymbol: string }) {
  // When linked from an embedded history/impact widget with ?symbol=..., the
  // user wants the focused symbol history, not the commit-pair picker. The
  // standalone panel already contains its own from/to selectors, so hide the
  // left commit sidebar to avoid duplicated controls.
  if (initialSymbol) {
    return <StandaloneSymbolHistory symbolId={initialSymbol} />;
  }

  const [selectedOld, setSelectedOld] = React.useState('');
  const [selectedNew, setSelectedNew] = React.useState('');
  const [modifiedCommits, setModifiedCommits] = React.useState<Set<string>>(new Set());

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
            const isModified = modifiedCommits.has(c.Hash);
            return (
              <li key={c.Hash} style={{ display: 'flex', gap: 6, alignItems: 'baseline', background: isModified ? 'rgba(255, 152, 0, 0.08)' : 'transparent', borderRadius: 4, padding: '2px 4px', margin: '-2px -4px' }}>
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
                  {isModified && <span style={{ marginLeft: 4, fontSize: 10, color: '#ff9800' }}>●</span>}
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
            <DiffView diff={diffQuery.data} initialSymbol={initialSymbol} oldHash={selectedOld} newHash={selectedNew} onModifiedCommitsChange={setModifiedCommits} />
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

function DiffView({ diff, initialSymbol, oldHash, newHash, onModifiedCommitsChange }: { diff: ReturnType<typeof useGetDiffQuery>['data'] & {}; initialSymbol: string; oldHash: string; newHash: string; onModifiedCommitsChange: (s: Set<string>) => void }) {
  if (!diff) return null;

  const [selectedSymbolId, setSelectedSymbolId] = React.useState(initialSymbol);

  // When a symbol is selected, fetch its history and compute which commits modified it
  const historyQuery = useGetSymbolHistoryQuery(
    { symbolId: selectedSymbolId },
    { skip: !selectedSymbolId },
  );

  React.useEffect(() => {
    if (!selectedSymbolId || !historyQuery.data) {
      onModifiedCommitsChange(new Set());
      return;
    }
    const modified = new Set<string>();
    const entries = historyQuery.data;
    for (let i = 0; i < entries.length; i++) {
      const prevHash = i < entries.length - 1 ? entries[i + 1].bodyHash : '';
      if (entries[i].bodyHash !== prevHash && entries[i].bodyHash !== '') {
        modified.add(entries[i].commitHash);
      }
    }
    onModifiedCommitsChange(modified);
    return () => onModifiedCommitsChange(new Set());
  }, [selectedSymbolId, historyQuery.data, onModifiedCommitsChange]);

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
                      <button
                        onClick={() => setSelectedSymbolId(selectedSymbolId === s.SymbolID ? '' : s.SymbolID)}
                        style={{
                          background: selectedSymbolId === s.SymbolID ? 'var(--cb-color-accent)' : 'transparent',
                          border: 'none',
                          color: selectedSymbolId === s.SymbolID ? '#fff' : 'inherit',
                          cursor: 'pointer',
                          padding: 0,
                          font: 'inherit',
                          textDecoration: 'none',
                        }}
                      >
                        <code>{s.Name}</code>
                      </button>
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

      {selectedSymbolId && (
        <section style={{ border: '1px solid var(--cb-color-accent)', borderRadius: 12, padding: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
            <h3 style={{ margin: 0 }}>Function diff</h3>
            <button
              onClick={() => setSelectedSymbolId('')}
              style={{ border: '1px solid var(--cb-color-border)', borderRadius: 4, padding: '2px 8px', cursor: 'pointer', background: 'transparent', color: 'var(--cb-color-text)' }}
            >
              Close
            </button>
          </div>
          <SymbolBodyDiffView from={oldHash} to={newHash} symbolId={selectedSymbolId} />
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

function StandaloneSymbolHistory({ symbolId }: { symbolId: string }) {
  const historyQuery = useGetSymbolHistoryQuery(
    { symbolId },
    { skip: !symbolId },
  );

  if (historyQuery.isLoading) return <div>Loading symbol history…</div>;
  if (historyQuery.error) return <div style={{ color: '#f44336' }}>Failed to load history: {JSON.stringify(historyQuery.error)}</div>;
  if (!historyQuery.data || historyQuery.data.length === 0) {
    return (
      <div>
        <h3 style={{ marginTop: 0 }}>Symbol history</h3>
        <p style={{ color: 'var(--cb-color-muted)' }}>No history found for <code>{symbolId}</code>.</p>
      </div>
    );
  }

  const name = symbolId.split('.').pop() || symbolId;
  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 12 }}>
        <h3 style={{ margin: 0 }}>History: <code>{name}</code></h3>
        <Link to="/history" style={{ fontSize: 12, color: 'var(--cb-color-link, #2196f3)' }}>← Back to commit diff</Link>
      </div>
      <p style={{ fontSize: 12, color: 'var(--cb-color-muted)', margin: '8px 0 0' }}>
        {symbolId}
      </p>
      <div style={{ marginTop: 16 }}>
        <SymbolHistoryPanel entries={historyQuery.data} symbolId={symbolId} />
      </div>
    </div>
  );
}

function SymbolHistoryPanel({ entries, symbolId }: { entries: SymbolHistoryEntry[]; symbolId: string }) {
  const [diffFrom, setDiffFrom] = React.useState('');
  const [diffTo, setDiffTo] = React.useState('');

  // Auto-select first and last entries with different body hashes
  React.useEffect(() => {
    if (entries.length >= 2 && !diffFrom && !diffTo) {
      setDiffTo(entries[0].commitHash);
      const firstDifferent = [...entries].reverse().find((e) => e.bodyHash !== entries[0].bodyHash && e.bodyHash !== '');
      setDiffFrom(firstDifferent?.commitHash ?? entries[entries.length - 1].commitHash);
    }
  }, [entries, diffFrom, diffTo]);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 8 }}>
        <div style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
          {entries.length} commit(s). Rows highlighted in orange = body changed.
          Select <strong>from</strong> and <strong>to</strong> to see the diff.
        </div>
      </div>
      <div style={{ overflowX: 'auto', marginBottom: 16 }}>
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Diff</th>
              <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Date</th>
              <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Hash</th>
              <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Lines</th>
              <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Body</th>
              <th style={{ textAlign: 'left', borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>Message</th>
            </tr>
          </thead>
          <tbody>
            {entries.map((e, i) => {
              const prevHash = i < entries.length - 1 ? entries[i + 1].bodyHash : '';
              const changed = e.bodyHash !== prevHash && e.bodyHash !== '';
              const date = new Date(e.authorTime * 1000);
              const dateStr = date.toISOString().slice(0, 16).replace('T', ' ');
              const isFrom = e.commitHash === diffFrom;
              const isTo = e.commitHash === diffTo;
              return (
                <tr key={i} style={{ background: changed ? 'rgba(255, 152, 0, 0.08)' : 'transparent' }}>
                  <td style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 4px', whiteSpace: 'nowrap' }}>
                    <button
                      onClick={() => setDiffFrom(isFrom ? '' : e.commitHash)}
                      style={{
                        padding: '0px 4px',
                        fontSize: 10,
                        border: isFrom ? '2px solid #2196f3' : '1px solid var(--cb-color-border)',
                        borderRadius: 3,
                        background: isFrom ? '#2196f3' : 'transparent',
                        color: isFrom ? '#fff' : 'inherit',
                        cursor: 'pointer',
                        marginRight: 2,
                      }}
                    >
                      from
                    </button>
                    <button
                      onClick={() => setDiffTo(isTo ? '' : e.commitHash)}
                      style={{
                        padding: '0px 4px',
                        fontSize: 10,
                        border: isTo ? '2px solid #4caf50' : '1px solid var(--cb-color-border)',
                        borderRadius: 3,
                        background: isTo ? '#4caf50' : 'transparent',
                        color: isTo ? '#fff' : 'inherit',
                        cursor: 'pointer',
                      }}
                    >
                      to
                    </button>
                  </td>
                  <td style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px', fontSize: 12, whiteSpace: 'nowrap' }}>
                    {dateStr}
                  </td>
                  <td style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>
                    <code style={{ fontSize: 12 }}>{e.shortHash}</code>
                  </td>
                  <td style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px', fontSize: 12 }}>
                    {e.startLine}-{e.endLine}
                  </td>
                  <td style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px' }}>
                    {changed ? (
                      <code style={{ fontSize: 11, color: '#ff9800', fontWeight: 700 }}>{e.bodyHash.slice(0, 7)}</code>
                    ) : (
                      <code style={{ fontSize: 11, color: 'var(--cb-color-muted)' }}>{e.bodyHash.slice(0, 7)}</code>
                    )}
                  </td>
                  <td style={{ borderBottom: '1px solid var(--cb-color-border)', padding: '6px 8px', fontSize: 12, maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {e.message}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {diffFrom && diffTo && diffFrom !== diffTo && (
        <SymbolBodyDiffView
          from={diffFrom}
          to={diffTo}
          symbolId={symbolId}
        />
      )}
    </div>
  );
}

function SymbolBodyDiffView({ from, to, symbolId }: { from: string; to: string; symbolId: string }) {
  const { data, isLoading, error } = useGetSymbolBodyDiffQuery(
    { from, to, symbolId },
    { skip: !from || !to },
  );

  if (isLoading) return <div style={{ padding: 16 }}>Loading body diff…</div>;
  if (error) return <div style={{ padding: 16, color: '#f44336' }}>Failed to load body diff: {JSON.stringify(error)}</div>;
  if (!data) return null;

  return (
    <div style={{ borderTop: '1px solid var(--cb-color-border)', paddingTop: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', marginBottom: 12 }}>
        <h3 style={{ margin: 0 }}>Body diff</h3>
        <div style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
          <code>{from.slice(0, 7)}</code> → <code>{to.slice(0, 7)}</code>
          {' '}({data.oldRange} → {data.newRange})
        </div>
      </div>

      {data.unifiedDiff ? (
        <pre
          style={{
            whiteSpace: 'pre-wrap',
            fontSize: 13,
            lineHeight: 1.5,
            background: 'var(--cb-color-surface, #f8f8f8)',
            border: '1px solid var(--cb-color-border)',
            borderRadius: 8,
            padding: 16,
            margin: 0,
            maxHeight: '50vh',
            overflow: 'auto',
          }}
        >
          {data.unifiedDiff.split('\n').map((line: string, i: number) => {
            if (line.startsWith('- ')) {
              return <div key={i} style={{ background: 'rgba(244, 67, 54, 0.12)', color: '#c62828' }}>{line}</div>;
            }
            if (line.startsWith('+ ')) {
              return <div key={i} style={{ background: 'rgba(76, 175, 80, 0.12)', color: '#2e7d32' }}>{line}</div>;
            }
            return <div key={i} style={{ color: 'var(--cb-color-muted)' }}>{line}</div>;
          })}
        </pre>
      ) : data.oldBody === data.newBody ? (
        <div style={{ padding: 16, color: 'var(--cb-color-muted)' }}>No body changes between these commits.</div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          <div>
            <div style={{ fontSize: 12, color: 'var(--cb-color-muted)', marginBottom: 4 }}>Old ({from.slice(0, 7)} {data.oldRange})</div>
            <pre
              style={{
                whiteSpace: 'pre-wrap',
                fontSize: 13,
                lineHeight: 1.5,
                background: 'var(--cb-color-surface, #f8f8f8)',
                border: '1px solid var(--cb-color-border)',
                borderRadius: 8,
                padding: 12,
                margin: 0,
                maxHeight: '40vh',
                overflow: 'auto',
              }}
            >
              {data.oldBody}
            </pre>
          </div>
          <div>
            <div style={{ fontSize: 12, color: 'var(--cb-color-muted)', marginBottom: 4 }}>New ({to.slice(0, 7)} {data.newRange})</div>
            <pre
              style={{
                whiteSpace: 'pre-wrap',
                fontSize: 13,
                lineHeight: 1.5,
                background: 'var(--cb-color-surface, #f8f8f8)',
                border: '1px solid var(--cb-color-border)',
                borderRadius: 8,
                padding: 12,
                margin: 0,
                maxHeight: '40vh',
                overflow: 'auto',
              }}
            >
              {data.newBody}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
