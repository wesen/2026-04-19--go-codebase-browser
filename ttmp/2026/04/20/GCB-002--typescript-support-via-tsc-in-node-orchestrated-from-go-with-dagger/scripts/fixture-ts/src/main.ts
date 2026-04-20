import { Greeter, greet, MaxRetries } from './greeter';

const g = new Greeter('Hello,');
console.log(g.hello('world'));
console.log(greet('typescript'));
console.log(`retries=${MaxRetries}`);
