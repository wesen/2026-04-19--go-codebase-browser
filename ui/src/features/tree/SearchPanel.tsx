import React from 'react';
import { Link } from 'react-router-dom';
import { useSearchSymbolsQuery } from '../../api/indexApi';
import { SearchBox } from '../../packages/ui/src/SearchBox';

export function SearchPanel() {
  const [q, setQ] = React.useState('');
  const { data, isFetching } = useSearchSymbolsQuery(
    { q },
    { skip: q.trim().length < 2 },
  );
  return (
    <div style={{ marginBottom: 16 }}>
      <SearchBox value={q} onChange={setQ} placeholder="Search symbols…" />
      {q.trim().length >= 2 && (
        <div>
          {isFetching && <div data-part="loading">Searching…</div>}
          {data && data.length === 0 && <div data-part="empty">No matches</div>}
          {data?.slice(0, 30).map((s) => (
            <Link
              key={s.id}
              to={`/symbol/${encodeURIComponent(s.id)}`}
              data-part="search-result"
              style={{ display: 'block', textDecoration: 'none', color: 'inherit' }}
            >
              <span data-part="symbol-kind" data-role={s.kind}>
                {s.kind}
              </span>{' '}
              <code data-part="symbol-name">{s.name}</code>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
