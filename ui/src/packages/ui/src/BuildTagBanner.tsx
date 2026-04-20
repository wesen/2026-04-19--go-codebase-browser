// React namespace provided by jsx: react-jsx
import { PARTS } from './parts';

export interface BuildTagBannerProps {
  tags: string[];
}

/**
 * BuildTagBanner renders the build constraints declared at the top of a Go
 * source file (e.g. `//go:build embed`). Only shown when non-empty.
 */
export function BuildTagBanner({ tags }: BuildTagBannerProps) {
  if (!tags.length) return null;
  return (
    <div data-part={PARTS.buildTagBanner} role="note">
      <strong>build:</strong>
      {tags.map((t, i) => (
        <code key={i} data-part={PARTS.buildTagChip}>{t}</code>
      ))}
    </div>
  );
}
