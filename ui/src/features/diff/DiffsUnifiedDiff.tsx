import React from 'react';
import type { DiffsUnifiedDiffRendererProps } from './DiffsUnifiedDiffRenderer';

const LazyDiffsUnifiedDiffRenderer = React.lazy(() =>
  import('./DiffsUnifiedDiffRenderer').then((mod) => ({ default: mod.DiffsUnifiedDiffRenderer })),
);

interface DiffsUnifiedDiffProps {
  name: string;
  oldText: string;
  newText: string;
  language?: string;
  oldLabel?: string;
  newLabel?: string;
  maxHeight?: string;
}

export function DiffsUnifiedDiff({
  name,
  oldText,
  newText,
  language = 'go',
  oldLabel,
  newLabel,
  maxHeight = '60vh',
}: DiffsUnifiedDiffProps) {
  const props: DiffsUnifiedDiffRendererProps = {
    name,
    oldText,
    newText,
    language,
    oldLabel,
    newLabel,
    maxHeight,
  };

  return (
    <DiffsErrorBoundary fallback={<FallbackUnifiedDiff oldText={oldText} newText={newText} maxHeight={maxHeight} />}>
      <React.Suspense fallback={<DiffLoading maxHeight={maxHeight} />}>
        <LazyDiffsUnifiedDiffRenderer {...props} />
      </React.Suspense>
    </DiffsErrorBoundary>
  );
}

class DiffsErrorBoundary extends React.Component<React.PropsWithChildren<{ fallback: React.ReactNode }>, { failed: boolean }> {
  state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch(error: unknown) {
    // Keep the page usable if the third-party diff renderer hits an edge case.
    console.warn('Diffs renderer failed; falling back to plain diff', error);
  }

  render() {
    return this.state.failed ? this.props.fallback : this.props.children;
  }
}

function DiffLoading({ maxHeight }: { maxHeight: string }) {
  return (
    <pre data-part="code-block" data-role="diff-loading" style={{ whiteSpace: 'pre-wrap', maxHeight, overflow: 'auto' }}>
      <code>Loading Diffs renderer…</code>
    </pre>
  );
}

function FallbackUnifiedDiff({ oldText, newText, maxHeight }: { oldText: string; newText: string; maxHeight: string }) {
  const lines = simpleUnifiedDiff(oldText, newText);
  return (
    <pre data-part="code-block" data-role="diff-fallback" style={{ whiteSpace: 'pre-wrap', maxHeight, overflow: 'auto' }}>
      <code>
        {lines.map((line, i) => {
          const style = line.startsWith('- ')
            ? { background: 'rgba(244, 67, 54, 0.12)', color: '#c62828', display: 'block' }
            : line.startsWith('+ ')
              ? { background: 'rgba(76, 175, 80, 0.12)', color: '#2e7d32', display: 'block' }
              : { color: 'var(--cb-color-muted)', display: 'block' };
          return <span key={i} style={style}>{line || ' '}</span>;
        })}
      </code>
    </pre>
  );
}

function simpleUnifiedDiff(oldText: string, newText: string): string[] {
  const oldLines = oldText.split('\n');
  const newLines = newText.split('\n');
  let prefix = 0;
  while (prefix < oldLines.length && prefix < newLines.length && oldLines[prefix] === newLines[prefix]) prefix++;
  let suffix = 0;
  while (
    suffix < oldLines.length - prefix &&
    suffix < newLines.length - prefix &&
    oldLines[oldLines.length - 1 - suffix] === newLines[newLines.length - 1 - suffix]
  ) suffix++;
  const out: string[] = [];
  for (let i = 0; i < prefix; i++) out.push(`  ${oldLines[i]}`);
  for (let i = prefix; i < oldLines.length - suffix; i++) out.push(`- ${oldLines[i]}`);
  for (let i = prefix; i < newLines.length - suffix; i++) out.push(`+ ${newLines[i]}`);
  for (let i = oldLines.length - suffix; i < oldLines.length; i++) out.push(`  ${oldLines[i]}`);
  return out;
}
