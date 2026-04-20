import { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react';
import { SearchBox } from '../SearchBox';

const meta: Meta<typeof SearchBox> = {
  title: 'Widgets/SearchBox',
  component: SearchBox,
};
export default meta;
type Story = StoryObj<typeof SearchBox>;

export const Default: Story = {
  render: () => {
    const [v, setV] = useState('');
    return <SearchBox value={v} onChange={setV} />;
  },
};

export const Prefilled: Story = {
  render: () => {
    const [v, setV] = useState('Build');
    return <SearchBox value={v} onChange={setV} />;
  },
};

export const CustomPlaceholder: Story = {
  render: () => {
    const [v, setV] = useState('');
    return <SearchBox value={v} onChange={setV} placeholder="Find a function…" />;
  },
};
