import type { Meta, StoryObj } from '@storybook/react';
import { SourceView } from '../SourceView';
import { sampleSource } from '../__fixtures__/symbols';

const meta: Meta<typeof SourceView> = {
  title: 'Widgets/SourceView',
  component: SourceView,
};
export default meta;
type Story = StoryObj<typeof SourceView>;

export const Default: Story = { args: { source: sampleSource } };

export const WithHighlight: Story = {
  args: { source: sampleSource, highlightLine: 4 },
};

export const Empty: Story = { args: { source: '' } };
