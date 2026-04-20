import React from 'react';
import { BrowserRouter, Link, Route, Routes, useNavigate } from 'react-router-dom';
import { widgetRootAttrs } from '../packages/ui/src/parts';
import '../packages/ui/src/theme/base.css';
import '../packages/ui/src/theme/dark.css';
import { HomePage } from '../features/tree/HomePage';
import { PackagePage } from '../features/tree/PackagePage';
import { SymbolPage } from '../features/symbol/SymbolPage';
import { SourcePage } from '../features/source/SourcePage';
import { SearchPanel } from '../features/tree/SearchPanel';
import { useGetIndexQuery } from '../api/indexApi';

export function App() {
  const [dark, setDark] = React.useState(false);
  return (
    <BrowserRouter>
      <div {...widgetRootAttrs} data-theme={dark ? 'dark' : 'light'}>
        <div data-part="layout">
          <aside data-part="sidebar">
            <Header onToggleTheme={() => setDark((d) => !d)} dark={dark} />
            <SearchPanel />
            <PackageList />
          </aside>
          <main data-part="main">
            <Routes>
              <Route path="/" element={<HomePage />} />
              <Route path="/packages/:id" element={<PackagePage />} />
              <Route path="/symbol/:id" element={<SymbolPage />} />
              <Route path="/source/*" element={<SourcePage />} />
            </Routes>
          </main>
        </div>
      </div>
    </BrowserRouter>
  );
}

function Header({ onToggleTheme, dark }: { onToggleTheme: () => void; dark: boolean }) {
  const { data } = useGetIndexQuery();
  return (
    <div style={{ marginBottom: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Link to="/" style={{ fontWeight: 700, color: 'var(--cb-color-accent)', textDecoration: 'none' }}>
          Codebase Browser
        </Link>
        <button
          onClick={onToggleTheme}
          style={{ background: 'transparent', border: '1px solid var(--cb-color-border)', padding: '2px 8px', borderRadius: 4, cursor: 'pointer', color: 'var(--cb-color-text)' }}
        >
          {dark ? '☀' : '☾'}
        </button>
      </div>
      {data && (
        <div data-part="symbol-doc" style={{ fontSize: 12, marginTop: 6 }}>
          {data.module} · {data.packages.length} pkg · {data.symbols.length} sym
        </div>
      )}
    </div>
  );
}

function PackageList() {
  const { data, isLoading, error } = useGetIndexQuery();
  const navigate = useNavigate();
  if (isLoading) return <div data-part="loading">Loading…</div>;
  if (error) return <div data-part="error">Failed to load index</div>;
  if (!data) return null;
  return (
    <ul data-part="tree-nav">
      {data.packages.map((p) => (
        <li key={p.id}>
          <a
            data-part="tree-node"
            href={`/packages/${encodeURIComponent(p.id)}`}
            onClick={(e) => {
              e.preventDefault();
              navigate(`/packages/${encodeURIComponent(p.id)}`);
            }}
          >
            {p.importPath.split('/').pop()}
            <div data-role="hint" style={{ fontSize: 11, color: 'var(--cb-color-muted)' }}>
              {p.importPath}
            </div>
          </a>
        </li>
      ))}
    </ul>
  );
}
