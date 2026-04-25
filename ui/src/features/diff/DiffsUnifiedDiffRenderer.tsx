import React from 'react';
import { MultiFileDiff } from '@pierre/diffs/react';
import type { FileContents } from '@pierre/diffs/react';

export interface DiffsUnifiedDiffRendererProps {
  name: string;
  oldText: string;
  newText: string;
  language: string;
  oldLabel?: string;
  newLabel?: string;
  maxHeight: string;
}

export function DiffsUnifiedDiffRenderer({
  name,
  oldText,
  newText,
  language,
  oldLabel,
  newLabel,
  maxHeight,
}: DiffsUnifiedDiffRendererProps) {
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
