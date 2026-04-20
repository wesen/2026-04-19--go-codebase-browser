// React namespace provided by jsx: react-jsx
import { useParams } from 'react-router-dom';
import { useGetDocQuery } from '../../api/docApi';

export function DocPage() {
  const { slug: rawSlug } = useParams<{ slug: string }>();
  const slug = rawSlug ?? '';
  const { data, isLoading, error } = useGetDocQuery(slug, { skip: !slug });
  if (isLoading) return <div data-part="loading">Loading…</div>;
  if (error) return <div data-part="error">Failed to load doc</div>;
  if (!data) return null;
  return (
    <article data-part="doc-page">
      {data.errors && data.errors.length > 0 && (
        <div data-part="error">
          {data.errors.map((e, i) => (
            <div key={i}>{e}</div>
          ))}
        </div>
      )}
      <div dangerouslySetInnerHTML={{ __html: data.html }} />
      <footer data-part="symbol-doc" style={{ marginTop: 32, fontSize: 12 }}>
        Resolved {data.snippets.length} snippet(s) from the live index.
      </footer>
    </article>
  );
}

export function DocList() {
  const { data } = useListDocs();
  if (!data?.length) return <div data-part="empty">No docs yet</div>;
  return (
    <ul data-part="tree-nav">
      {data.map((d) => (
        <li key={d.slug}>
          <a data-part="tree-node" href={`/doc/${encodeURIComponent(d.slug)}`}>
            {d.title}
          </a>
        </li>
      ))}
    </ul>
  );
}

// Narrow re-export to avoid an unused-import warning in this file.
import { useListDocsQuery } from '../../api/docApi';
const useListDocs = useListDocsQuery;
