import type { Meta, StoryObj } from '@storybook/react';
import { Code } from '../Code';
import { SymbolCard } from '../SymbolCard';
import { BuildTagBanner } from '../BuildTagBanner';
import type { Symbol } from '../../../../api/types';

const meta: Meta = {
  title: 'Widgets/Annotations',
};
export default meta;
type Story = StoryObj;

const annotated = `// Old computes the old sum.
//
// Deprecated: use NewSum instead. This will be removed in v2.0.
func Old(a, b int) int {
    // TODO: support negative numbers
    // FIXME: overflow for very large ints
    // NOTE: this is called from the legacy pipeline
    // HACK: skip the first entry
    // BUG(wesen): off-by-one for empty slices
    return a + b
}`;

export const CommentAnnotationsInCode: Story = {
  render: () => <Code text={annotated} />,
};

const deprecatedSymbol: Symbol = {
  id: 'sym:example.com/foo.func.OldGreet',
  kind: 'func',
  name: 'OldGreet',
  packageId: 'pkg:example.com/foo',
  fileId: 'file:foo.go',
  range: { startLine: 1, startCol: 1, endLine: 3, endCol: 2, startOffset: 0, endOffset: 60 },
  doc: 'Deprecated: use Greet instead. OldGreet remains for backward compatibility.',
  signature: 'func OldGreet(name string) string',
  exported: true,
};

export const DeprecatedSymbolCard: Story = {
  render: () => <SymbolCard symbol={deprecatedSymbol} snippet="func OldGreet(name string) string { return name }" />,
};

export const BuildTagBannerDefault: Story = {
  render: () => <BuildTagBanner tags={['embed', 'linux && amd64']} />,
};

export const BuildTagBannerHidden: Story = {
  render: () => (
    <div>
      <em>Banner is not rendered when the tag list is empty:</em>
      <BuildTagBanner tags={[]} />
      <em>(no output above)</em>
    </div>
  ),
};
