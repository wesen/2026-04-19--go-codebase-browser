// React namespace provided by jsx: react-jsx
import { Link, useParams } from 'react-router-dom';
import { useGetSymbolQuery } from '../../api/indexApi';
import { useGetSnippetQuery } from '../../api/sourceApi';
import { SymbolCard } from '../../packages/ui/src/SymbolCard';

export function SymbolPage() {
  const { id: rawId } = useParams<{ id: string }>();
  const id = rawId ? decodeURIComponent(rawId) : '';
  const { data: sym, isLoading, error } = useGetSymbolQuery(id, { skip: !id });
  const { data: snippet } = useGetSnippetQuery(
    { sym: id, kind: 'declaration' },
    { skip: !id },
  );
  if (isLoading) return <div data-part="loading">Loading…</div>;
  if (error) return <div data-part="error">Failed to load symbol</div>;
  if (!sym) return <div data-part="empty">Symbol not found</div>;

  return (
    <div>
      <div data-part="symbol-doc" style={{ fontSize: 12 }}>
        <Link to={`/packages/${encodeURIComponent(sym.packageId)}`}>
          {sym.packageId.replace(/^pkg:/, '')}
        </Link>
      </div>
      <SymbolCard symbol={sym} snippet={snippet} />
      <p data-part="symbol-doc">
        File:{' '}
        <Link to={`/source/${sym.fileId.replace(/^file:/, '')}`}>
          {sym.fileId.replace(/^file:/, '')}
        </Link>{' '}
        (lines {sym.range.startLine}–{sym.range.endLine})
      </p>
    </div>
  );
}
