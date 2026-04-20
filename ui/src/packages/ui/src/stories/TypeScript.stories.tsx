import type { Meta, StoryObj } from '@storybook/react';
import { Code } from '../Code';
import { SymbolCard } from '../SymbolCard';
import type { Symbol } from '../../../../api/types';

const meta: Meta = { title: 'Widgets/TypeScript' };
export default meta;
type Story = StoryObj;

const tsSnippet = `import type { ReactNode } from 'react';

/** Greeter produces greetings with a configurable prefix. */
export class Greeter {
  constructor(public readonly prefix: string) {}

  async hello(name: string): Promise<string> {
    // TODO: support localisation
    return \`\${this.prefix} \${name}!\`;
  }
}

export interface Greetable<T = string> {
  greet(name: T): Promise<T>;
}

export const MaxRetries = 3 as const;
export type Prefix = string | undefined;`;

export const CodeBlockTS: Story = {
  render: () => <Code text={tsSnippet} language="ts" />,
};

const classSymbol: Symbol = {
  id: 'sym:ui/src/packages/ui/src/stories/TypeScript.class.Greeter',
  kind: 'class',
  name: 'Greeter',
  packageId: 'pkg:ui/src/packages/ui/src/stories',
  fileId: 'file:ui/src/packages/ui/src/stories/TypeScript.tsx',
  range: { startLine: 1, startCol: 1, endLine: 9, endCol: 2, startOffset: 0, endOffset: 220 },
  doc: 'Greeter produces greetings with a configurable prefix.',
  signature: 'class Greeter',
  exported: true,
  language: 'ts',
};

const methodSymbol: Symbol = {
  id: 'sym:ui/src/packages/ui/src/stories/TypeScript.method.Greeter.hello',
  kind: 'method',
  name: 'hello',
  packageId: 'pkg:ui/src/packages/ui/src/stories',
  fileId: 'file:ui/src/packages/ui/src/stories/TypeScript.tsx',
  range: { startLine: 5, startCol: 3, endLine: 8, endCol: 4, startOffset: 130, endOffset: 210 },
  signature: 'async hello(name: string): Promise<string>',
  doc: 'Greets name with the configured prefix.',
  receiver: { typeName: 'Greeter', pointer: false },
  exported: true,
  language: 'ts',
};

const ifaceSymbol: Symbol = {
  id: 'sym:ui/src/packages/ui/src/stories/TypeScript.iface.Greetable',
  kind: 'iface',
  name: 'Greetable',
  packageId: 'pkg:ui/src/packages/ui/src/stories',
  fileId: 'file:ui/src/packages/ui/src/stories/TypeScript.tsx',
  range: { startLine: 11, startCol: 1, endLine: 13, endCol: 2, startOffset: 230, endOffset: 290 },
  signature: 'interface Greetable<T = string>',
  exported: true,
  language: 'ts',
};

const aliasSymbol: Symbol = {
  id: 'sym:ui/src/packages/ui/src/stories/TypeScript.alias.Prefix',
  kind: 'alias',
  name: 'Prefix',
  packageId: 'pkg:ui/src/packages/ui/src/stories',
  fileId: 'file:ui/src/packages/ui/src/stories/TypeScript.tsx',
  range: { startLine: 15, startCol: 1, endLine: 15, endCol: 40, startOffset: 300, endOffset: 340 },
  signature: 'type Prefix = string | undefined',
  exported: true,
  language: 'ts',
};

export const ClassCard: Story = {
  render: () => <SymbolCard symbol={classSymbol} snippet={`class Greeter {\n  constructor(public readonly prefix: string) {}\n}`} />,
};

export const MethodCard: Story = {
  render: () => <SymbolCard symbol={methodSymbol} snippet={`async hello(name: string): Promise<string> {\n  return \`\${this.prefix} \${name}!\`;\n}`} />,
};

export const InterfaceCard: Story = {
  render: () => <SymbolCard symbol={ifaceSymbol} snippet={`interface Greetable<T = string> {\n  greet(name: T): Promise<T>;\n}`} />,
};

export const AliasCard: Story = {
  render: () => <SymbolCard symbol={aliasSymbol} snippet={`type Prefix = string | undefined`} />,
};

// Showcase the JSX polish: component names (capitalized) after `<` / `</`
// tint as type while lowercase DOM tags (div, span) keep the id colour.
const jsxSnippet = `import { useState } from 'react';
import { Greeter } from './greeter';

interface Props { name: string }

export function HelloCard({ name }: Props) {
  const [open, setOpen] = useState(true);
  return (
    <div className="card" role="region">
      <Greeter.Label>Hello</Greeter.Label>
      <button onClick={() => setOpen((v) => !v)}>
        {open ? 'Hide' : <Icon name="chevron" />}
      </button>
      {open && <p>Welcome, {name}.</p>}
    </div>
  );
}`;

export const CodeBlockJSX: Story = {
  render: () => <Code text={jsxSnippet} language="tsx" />,
};
