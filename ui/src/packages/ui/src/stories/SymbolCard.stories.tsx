import type { Meta, StoryObj } from '@storybook/react';
import { SymbolCard } from '../SymbolCard';
import {
  funcSymbol,
  methodSymbol,
  structSymbol,
  ifaceSymbol,
  constSymbol,
  sampleSnippet,
} from '../__fixtures__/symbols';

const meta: Meta<typeof SymbolCard> = {
  title: 'Widgets/SymbolCard',
  component: SymbolCard,
};
export default meta;

type Story = StoryObj<typeof SymbolCard>;

export const Default: Story = { args: { symbol: funcSymbol } };

export const WithSnippet: Story = {
  args: { symbol: funcSymbol, snippet: sampleSnippet },
};

export const MethodKind: Story = { args: { symbol: methodSymbol } };

export const StructKind: Story = { args: { symbol: structSymbol, snippet: structSymbol.signature } };

export const InterfaceKind: Story = { args: { symbol: ifaceSymbol, snippet: ifaceSymbol.signature } };

export const ConstKind: Story = { args: { symbol: constSymbol } };

export const NoDoc: Story = {
  args: { symbol: { ...funcSymbol, doc: undefined } },
};

export const WithNameRenderer: Story = {
  args: {
    symbol: funcSymbol,
    renderName: (name: string) => <a href="#">{name}</a>,
  },
};

export const AllKinds: Story = {
  render: () => (
    <>
      {[funcSymbol, methodSymbol, structSymbol, ifaceSymbol, constSymbol].map((s) => (
        <SymbolCard key={s.id} symbol={s} />
      ))}
    </>
  ),
};
