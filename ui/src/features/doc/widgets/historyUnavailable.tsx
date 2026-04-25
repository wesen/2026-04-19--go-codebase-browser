export function isHistoryUnavailable(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false;
  const maybe = error as { status?: unknown; data?: unknown };
  if (maybe.status === 404 || maybe.status === 503) return true;
  if (typeof maybe.data === 'object' && maybe.data !== null) {
    const data = maybe.data as { error?: unknown; message?: unknown };
    const text = String(data.error ?? data.message ?? '').toLowerCase();
    return text.includes('history database') || text.includes('history db') || text.includes('history api');
  }
  return false;
}

export function HistoryUnavailableNotice({ widget }: { widget: string }) {
  return (
    <section data-part="doc-snippet" data-role="history-unavailable">
      <div
        style={{
          border: '1px dashed var(--cb-color-border)',
          borderRadius: 10,
          padding: 12,
          background: 'rgba(255, 193, 7, 0.08)',
        }}
      >
        <strong>{widget} needs history data.</strong>
        <p style={{ margin: '6px 0 0', color: 'var(--cb-color-muted)' }}>
          This published server was started without a history database, so the
          history-backed demo cannot render here. Run locally with{' '}
          <code>--history-db history.db --repo-root .</code> to enable semantic
          diffs, symbol timelines, and impact analysis.
        </p>
      </div>
    </section>
  );
}
