// React namespace provided by jsx: react-jsx
import { Link, useLocation, useSearchParams } from 'react-router-dom';
import { useGetSourceQuery, useGetSourceRefsQuery } from '../../api/sourceApi';
import { useGetIndexQuery } from '../../api/indexApi';
import { SourceView } from '../../packages/ui/src/SourceView';
import { BuildTagBanner } from '../../packages/ui/src/BuildTagBanner';
import { FileXrefPanel } from './FileXrefPanel';

export function SourcePage() {
  const location = useLocation();
  const [params] = useSearchParams();
  const path = decodeURIComponent(location.pathname.replace(/^\/source\//, ''));
  const highlight = Number(params.get('line') ?? '0') || undefined;

  const { data, isLoading, error } = useGetSourceQuery(path, { skip: !path });
  // File-level xrefs for linkified identifiers. Skipped until we have a
  // path so the request isn't fired during the initial undefined render.
  const { data: refs } = useGetSourceRefsQuery(path, { skip: !path });
  // Build tags come from the index, not the raw file, so they're a cheap
  // derived lookup.
  const { data: index } = useGetIndexQuery();
  const file = index?.files.find((f) => f.path === path);
  const tags = file?.buildTags ?? [];

  if (isLoading) return <div data-part="loading">Loading…</div>;
  if (error) return <div data-part="error">Failed to load source</div>;
  if (!data) return <div data-part="empty">Empty file</div>;
  return (
    <div>
      <h2 style={{ marginTop: 0 }}>{path}</h2>
      <BuildTagBanner tags={tags} />
      <SourceView
        source={data}
        highlightLine={highlight}
        language={file?.language ?? 'go'}
        refs={refs}
        renderRefLink={(symbolId, children) => (
          <Link to={`/symbol/${encodeURIComponent(symbolId)}`} data-role="xref">
            {children}
          </Link>
        )}
      />
      <FileXrefPanel path={path} />
    </div>
  );
}
