import type { Meta, StoryObj } from '@storybook/react';
import { Code } from '../Code';

const meta: Meta<typeof Code> = {
  title: 'Widgets/Code',
  component: Code,
};
export default meta;
type Story = StoryObj<typeof Code>;

const goSnippet = `// Greet returns a greeting for name.
func Greet(name string) string {
    if name == "" {
        name = "world"
    }
    return "Hello, " + name + "!"
}`;

const goWithStrings = `package foo

const banner = \`
multiline raw
string literal
\`

var max = 3.14
var r rune = 'x'
var greeting = "hello\\tworld\\n"`;

const goInterface = `type Greetable interface {
    Greet(name string) string
    Farewell() error
}`;

const goStruct = `type Greeter struct {
    Prefix string // honorific
    count  int    // unexported
}`;

export const Default: Story = { args: { text: goSnippet } };
export const WithStringLiterals: Story = { args: { text: goWithStrings } };
export const Interface: Story = { args: { text: goInterface } };
export const StructWithComments: Story = { args: { text: goStruct } };

export const UnknownLanguage: Story = {
  args: { text: 'SELECT * FROM users WHERE id = ?', language: 'sql' },
};
