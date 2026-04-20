// React namespace provided by jsx: react-jsx
import { useLocation, useSearchParams } from 'react-router-dom';
import { useGetSourceQuery } from '../../api/sourceApi';
import { SourceView } from '../../packages/ui/src/SourceView';

export function SourcePage() {
  const location = useLocation();
  const [params] = useSearchParams();
  // /source/<relpath>
  const path = decodeURIComponent(location.pathname.replace(/^\/source\//, ''));
  const highlight = Number(params.get('line') ?? '0') || undefined;

  const { data, isLoading, error } = useGetSourceQuery(path, { skip: !path });
  if (isLoading) return <div data-part="loading">Loading…</div>;
  if (error) return <div data-part="error">Failed to load source</div>;
  if (!data) return <div data-part="empty">Empty file</div>;
  return (
    <div>
      <h2 style={{ marginTop: 0 }}>{path}</h2>
      <SourceView source={data} highlightLine={highlight} />
    </div>
  );
}
