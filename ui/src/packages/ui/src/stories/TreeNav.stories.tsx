import type { Meta, StoryObj } from '@storybook/react';
import { TreeNav } from '../TreeNav';

const meta: Meta<typeof TreeNav> = {
  title: 'Widgets/TreeNav',
  component: TreeNav,
};
export default meta;
type Story = StoryObj<typeof TreeNav>;

export const Default: Story = {
  args: {
    items: [
      { id: '1', label: 'cmd/codebase-browser', hint: '4 files' },
      { id: '2', label: 'internal/browser', hint: '1 file' },
      { id: '3', label: 'internal/indexer', hint: '4 files' },
      { id: '4', label: 'internal/staticapp', hint: '5 files' },
      { id: '5', label: 'internal/sourcefs', hint: '2 files' },
    ],
  },
};

export const Empty: Story = { args: { items: [] } };

export const WithActive: Story = {
  args: {
    items: [
      { id: '1', label: 'cmd/codebase-browser' },
      { id: '2', label: 'internal/browser', active: true },
      { id: '3', label: 'internal/staticapp' },
    ],
  },
};
