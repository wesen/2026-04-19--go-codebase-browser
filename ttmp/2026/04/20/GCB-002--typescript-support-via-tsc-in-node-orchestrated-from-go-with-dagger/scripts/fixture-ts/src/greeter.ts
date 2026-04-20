/** Greeter produces greetings with a configurable prefix. */
export class Greeter {
  constructor(public prefix: string) {}

  /** Hello returns a greeting for name. */
  hello(name: string): string {
    return `${this.prefix} ${name}`;
  }
}

/** MaxRetries bounds retry attempts. */
export const MaxRetries = 3;

/** A greeting function. */
export function greet(name: string): string {
  return `Hello, ${name}!`;
}

export interface Greetable {
  greet(name: string): string;
}

export type Prefix = string;
