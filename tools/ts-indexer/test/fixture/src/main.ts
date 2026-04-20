import { greet, Greeter, MaxRetries } from './greeter.js';

export function run(): number {
  const g = new Greeter('Hi');
  g.hello('world');
  greet('world');
  return MaxRetries;
}
