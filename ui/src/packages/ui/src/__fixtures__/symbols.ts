import type { Symbol } from '../../../../api/types';

export const funcSymbol: Symbol = {
  id: 'sym:example.com/foo.func.Greet',
  kind: 'func',
  name: 'Greet',
  packageId: 'pkg:example.com/foo',
  fileId: 'file:foo.go',
  range: { startLine: 12, startCol: 1, endLine: 15, endCol: 2, startOffset: 100, endOffset: 170 },
  signature: 'func Greet(name string) string',
  doc: 'Greet returns a greeting for name. If name is empty it greets the world.',
  exported: true,
};

export const methodSymbol: Symbol = {
  id: 'sym:example.com/foo.method.Greeter.Hello',
  kind: 'method',
  name: 'Hello',
  packageId: 'pkg:example.com/foo',
  fileId: 'file:greeter.go',
  range: { startLine: 20, startCol: 1, endLine: 24, endCol: 2, startOffset: 200, endOffset: 260 },
  signature: 'func (g *Greeter) Hello(name string) string',
  doc: 'Hello returns a prefixed greeting.',
  receiver: { typeName: 'Greeter', pointer: true },
  exported: true,
};

export const structSymbol: Symbol = {
  id: 'sym:example.com/foo.struct.Greeter',
  kind: 'struct',
  name: 'Greeter',
  packageId: 'pkg:example.com/foo',
  fileId: 'file:greeter.go',
  range: { startLine: 5, startCol: 1, endLine: 8, endCol: 2, startOffset: 50, endOffset: 90 },
  signature: 'type Greeter struct {\n\tPrefix string\n}',
  doc: 'Greeter produces greetings with a configurable prefix.',
  exported: true,
};

export const ifaceSymbol: Symbol = {
  id: 'sym:example.com/foo.iface.Greetable',
  kind: 'iface',
  name: 'Greetable',
  packageId: 'pkg:example.com/foo',
  fileId: 'file:greetable.go',
  range: { startLine: 3, startCol: 1, endLine: 5, endCol: 2, startOffset: 10, endOffset: 60 },
  signature: 'type Greetable interface {\n\tGreet(name string) string\n}',
  doc: 'Greetable describes anything that can greet.',
  exported: true,
};

export const constSymbol: Symbol = {
  id: 'sym:example.com/foo.const.MaxRetries',
  kind: 'const',
  name: 'MaxRetries',
  packageId: 'pkg:example.com/foo',
  fileId: 'file:foo.go',
  range: { startLine: 2, startCol: 1, endLine: 2, endCol: 22, startOffset: 5, endOffset: 30 },
  signature: 'const MaxRetries = 3',
  doc: 'MaxRetries bounds retry attempts.',
  exported: true,
};

export const sampleSnippet = `func Greet(name string) string {
    if name == "" {
        name = "world"
    }
    return "Hello, " + name + "!"
}`;

export const sampleSource = `package foo

// Greet returns a greeting for name.
func Greet(name string) string {
    if name == "" {
        name = "world"
    }
    return "Hello, " + name + "!"
}
`;
