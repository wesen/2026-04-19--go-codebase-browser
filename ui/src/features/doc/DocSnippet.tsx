// React namespace provided by jsx: react-jsx
import React from 'react';
import { Link } from 'react-router-dom';
import { useGetSymbolQuery } from '../../api/indexApi';
import { ExpandableSymbol } from '../symbol/ExpandableSymbol';
import { XrefPanel } from '../symbol/XrefPanel';
import { Code } from '../../packages/ui/src/Code';
import { SymbolDiffInlineWidget } from './widgets/SymbolDiffInlineWidget';

/**
 * useGetSnippetFromCommit fetches a symbol's snippet at a specific commit
 * from the history API. Returns undefined while loading, null on error,
 * or the snippet text string on success. When commit is undefined, returns
 * null immediately (used as a "skip" signal).
 */
function useGetSnippetFromCommit(
  sym: string,
  kind: string,
  commit?: string,
): string | null | undefined {
  // No commit specified → not a history-aware request.
  if (!commit) return null;

  // We use a synchronous fetch pattern to keep the hook simple.
  // RTK-Query with dynamic base URL would be cleaner, but for Slice 0
  // this direct fetch keeps the blast radius minimal.
  const cache = React.useRef<Map<string, string | null>>(new Map());
  const key = `${sym}|${kind}|${commit}`;
  const [, forceUpdate] = React.useReducer((x: number) => x + 1, 0);

  React.useEffect(() => {
    if (cache.current.has(key)) return;
    // Mark as "loading" by setting undefined (not in cache).
    // We use a sentinel to track in-flight requests.
    const controller = new AbortController();
    const url = `/api/snippet?sym=${encodeURIComponent(sym)}&kind=${encodeURIComponent(kind)}&commit=${encodeURIComponent(commit)}`;
    fetch(url, { signal: controller.signal })
      .then((resp) => {
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        return resp.text();
      })
      .then((text) => {
        cache.current.set(key, text);
        forceUpdate();
      })
      .catch(() => {
        cache.current.set(key, null);
        forceUpdate();
      });
    return () => controller.abort();
  }, [key, sym, kind, commit]);

  if (!cache.current.has(key)) return undefined; // loading
  return cache.current.get(key);
}

/**
 * DocSnippet hydrates one `[data-codebase-snippet]` stub on a doc page.
 * The server-rendered stub carries the symbol id + directive type; we
 * dispatch to the right widget so every directive on a doc page gets
 * the same interactive treatment a /symbol/{id} page does:
 *
 *   - codebase-snippet    → <LinkedCode> with clickable xrefs
 *   - codebase-signature  → compact <Link> to the symbol
 *   - codebase-doc        → blockquote of the godoc
 */
export interface DocSnippetProps {
  sym: string;
  directive: string;
  kind: string;
  lang: string;
  commit?: string;
  params?: Record<string, string>;
}

export function DocSnippet({ sym, directive, lang, commit, params }: DocSnippetProps) {
  if (directive === 'codebase-diff') {
    return <SymbolDiffInlineWidget sym={sym} from={params?.from ?? ''} to={params?.to ?? ''} />;
  }
  if (directive === 'codebase-signature') return <DocSignature sym={sym} commit={commit} language={lang} />;
  if (directive === 'codebase-doc') return <DocGodoc sym={sym} commit={commit} />;
  return <DocFullSnippet sym={sym} commit={commit} language={lang} />;
}

function DocSignature({ sym, commit, language }: { sym: string; commit?: string; language?: string }) {
  const { data } = useGetSymbolQuery(sym);
  const snippet = useGetSnippetFromCommit(sym, 'signature', commit);
  const display = commit ? (snippet ?? data?.signature ?? data?.name ?? sym) : (data?.signature ?? data?.name ?? sym);
  if (commit) {
    return <Code text={display} language={language || 'go'} />;
  }
  return (
    <pre data-part="code-block" data-role="signature">
      <Link to={`/symbol/${encodeURIComponent(sym)}`} data-role="xref">
        <code data-tok="kw">{display}</code>
      </Link>
    </pre>
  );
}

function DocGodoc({ sym, commit: _commit }: { sym: string; commit?: string }) {
  const { data } = useGetSymbolQuery(sym);
  // Doc comments don't change often; use static index for now.
  // History-backed doc resolution can be added later.
  return (
    <blockquote data-part="symbol-doc" data-role="doc">
      {data?.doc ?? ''}
    </blockquote>
  );
}

// DocFullSnippet wraps <ExpandableSymbol> (same component used on symbol
// pages) and adds a collapsible cross-reference panel beneath it. Readers
// get the same rich, navigable view of an embedded snippet that they'd see
// on /symbol/{id} — plus a show/hide toggle so long snippets can collapse
// once they've been skimmed, and a <details> section for digging into
// callers and callees without leaving the doc page.
function DocFullSnippet({ sym, commit, language }: { sym: string; commit?: string; language?: string }) {
  const { data: symbol } = useGetSymbolQuery(sym);
  const commitSnippet = useGetSnippetFromCommit(sym, 'declaration', commit);

  if (commit && commitSnippet === undefined) {
    return (
      <pre data-part="code-block">
        <code>Loading snippet at commit {commit.slice(0, 7)}…</code>
      </pre>
    );
  }
  if (!symbol && !commit) {
    return (
      <pre data-part="code-block">
        <code>Loading…</code>
      </pre>
    );
  }

  // When commit is set, render the commit-resolved snippet as a simple
  // code block (without the full ExpandableSymbol treatment since xrefs
  // are not available for non-HEAD commits yet).
  if (commit && commitSnippet) {
    return (
      <section data-part="doc-snippet">
        <div style={{ fontSize: 12, color: 'var(--cb-color-muted)', marginBottom: 8 }}>
          at commit <code>{commit.slice(0, 7)}</code>
        </div>
        <Code text={commitSnippet} language={language || symbol?.language || 'go'} />
      </section>
    );
  }

  return (
    <section data-part="doc-snippet">
      <ExpandableSymbol symbol={symbol!} defaultOpen />
      <details data-part="doc-snippet-xref" style={{ marginTop: 8 }}>
        <summary data-role="hint" style={{ cursor: 'pointer' }}>
          cross-references
        </summary>
        <XrefPanel symbolId={sym} />
      </details>
    </section>
  );
}
