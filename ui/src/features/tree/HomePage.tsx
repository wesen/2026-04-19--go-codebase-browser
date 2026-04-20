// React namespace provided by jsx: react-jsx
import { useGetIndexQuery } from '../../api/indexApi';

export function HomePage() {
  const { data, isLoading } = useGetIndexQuery();
  if (isLoading) return <div data-part="loading">Loading…</div>;
  if (!data) return null;
  return (
    <div>
      <h1 style={{ marginTop: 0 }}>{data.module}</h1>
      <div data-part="symbol-doc">
        Go version: {data.goVersion} · Generated: {data.generatedAt}
      </div>
      <dl style={{ display: 'grid', gridTemplateColumns: 'auto 1fr', gap: '4px 16px', marginTop: 24 }}>
        <dt>Packages</dt>
        <dd>{data.packages.length}</dd>
        <dt>Files</dt>
        <dd>{data.files.length}</dd>
        <dt>Symbols</dt>
        <dd>{data.symbols.length}</dd>
      </dl>
      <p style={{ marginTop: 24, color: 'var(--cb-color-muted)' }}>
        Pick a package on the left to drill in, or use search to find a symbol.
      </p>
    </div>
  );
}
