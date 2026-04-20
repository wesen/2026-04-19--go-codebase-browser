import type { Meta, StoryObj } from '@storybook/react';
import { Snippet } from '../Snippet';
import { sampleSnippet } from '../__fixtures__/symbols';

const meta: Meta<typeof Snippet> = {
  title: 'Widgets/Snippet',
  component: Snippet,
};
export default meta;
type Story = StoryObj<typeof Snippet>;

export const Default: Story = { args: { text: sampleSnippet } };

export const WithJumpLink: Story = {
  args: {
    text: sampleSnippet,
    jumpTo: '/symbol/sym%3Aexample.com%2Ffoo.func.Greet',
  },
};

export const SingleLine: Story = { args: { text: 'const MaxRetries = 3' } };

export const LongSnippet: Story = {
  args: {
    text: Array.from({ length: 30 }, (_, i) => `line ${i + 1}: println("${i}")`).join('\n'),
  },
};
