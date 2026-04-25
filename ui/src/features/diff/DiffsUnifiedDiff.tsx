import React from 'react';
import { MultiFileDiff } from '@pierre/diffs/react';
import type { FileContents } from '@pierre/diffs/react';

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
  const [diffStyle, setDiffStyle] = React.useState<'unified' | 'split'>('unified');
  const oldFile = React.useMemo<FileContents>(() => ({
    name: oldLabel ? `${name} (${oldLabel})` : name,
    contents: oldText,
    lang: language,
    cacheKey: oldLabel ? `${name}:${oldLabel}` : undefined,
  }), [language, name, oldLabel, oldText]);

  const newFile = React.useMemo<FileContents>(() => ({
    name: newLabel ? `${name} (${newLabel})` : name,
    contents: newText,
    lang: language,
    cacheKey: newLabel ? `${name}:${newLabel}` : undefined,
  }), [language, name, newLabel, newText]);

  return (
    <DiffsErrorBoundary fallback={<FallbackUnifiedDiff oldText={oldText} newText={newText} />}>
      <div data-role="diffs-unified-diff">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, marginBottom: 8, flexWrap: 'wrap' }}>
          <span style={{ fontSize: 12, color: 'var(--cb-color-muted)' }}>
            Rendered with Diffs · word-level changes enabled
          </span>
          <div role="group" aria-label="Diff layout" style={{ display: 'inline-flex', border: '1px solid var(--cb-color-border)', borderRadius: 999, overflow: 'hidden' }}>
            <button
              type="button"
              onClick={() => setDiffStyle('unified')}
              aria-pressed={diffStyle === 'unified'}
              style={toggleButtonStyle(diffStyle === 'unified')}
            >
              Unified
            </button>
            <button
              type="button"
              onClick={() => setDiffStyle('split')}
              aria-pressed={diffStyle === 'split'}
              style={toggleButtonStyle(diffStyle === 'split')}
            >
              Split
            </button>
          </div>
        </div>
        <div style={{ maxHeight, overflow: 'auto' }}>
          <MultiFileDiff
            oldFile={oldFile}
            newFile={newFile}
            disableWorkerPool
            options={{
              diffStyle,
              overflow: 'wrap',
              theme: { light: 'github-light', dark: 'github-dark' },
              themeType: 'system',
              hunkSeparators: 'line-info-basic',
              lineDiffType: 'word',
            }}
          />
        </div>
      </div>
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

function FallbackUnifiedDiff({ oldText, newText }: { oldText: string; newText: string }) {
  const lines = simpleUnifiedDiff(oldText, newText);
  return (
    <pre data-part="code-block" data-role="diff-fallback" style={{ whiteSpace: 'pre-wrap', maxHeight: '60vh', overflow: 'auto' }}>
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

function toggleButtonStyle(active: boolean): React.CSSProperties {
  return {
    border: 0,
    borderRight: active ? 0 : '1px solid var(--cb-color-border)',
    background: active ? 'var(--cb-color-accent, #2196f3)' : 'transparent',
    color: active ? '#fff' : 'var(--cb-color-text)',
    padding: '4px 10px',
    cursor: 'pointer',
    fontSize: 12,
    fontWeight: active ? 700 : 400,
  };
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
