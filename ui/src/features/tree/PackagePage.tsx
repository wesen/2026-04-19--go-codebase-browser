// React namespace provided by jsx: react-jsx
import { Link, useParams } from 'react-router-dom';
import { useGetIndexQuery } from '../../api/indexApi';
import { SymbolCard } from '../../packages/ui/src/SymbolCard';

export function PackagePage() {
  const { id: rawId } = useParams<{ id: string }>();
  const id = rawId ? decodeURIComponent(rawId) : undefined;
  const { data } = useGetIndexQuery();
  if (!data) return <div data-part="loading">Loading…</div>;

  const pkg = data.packages.find((p) => p.id === id);
  if (!pkg) return <div data-part="empty">Package not found</div>;

  const symbols = data.symbols.filter((s) => s.packageId === pkg.id);
  const files = data.files.filter((f) => f.packageId === pkg.id);

  return (
    <div>
      <h1 style={{ marginTop: 0 }}>{pkg.importPath}</h1>
      {pkg.doc && <p data-part="symbol-doc">{pkg.doc}</p>}

      <h2>Files ({files.length})</h2>
      <ul data-part="tree-nav">
        {files.map((f) => (
          <li key={f.id}>
            <Link data-part="tree-node" to={`/source/${f.path}`}>
              {f.path} <span data-role="hint">· {f.lineCount} lines</span>
            </Link>
          </li>
        ))}
      </ul>

      <h2>Symbols ({symbols.length})</h2>
      {symbols.map((s) => (
        <SymbolCard
          key={s.id}
          symbol={s}
          renderName={(name, sid) => (
            <Link to={`/symbol/${encodeURIComponent(sid)}`}>{name}</Link>
          )}
        />
      ))}
    </div>
  );
}
