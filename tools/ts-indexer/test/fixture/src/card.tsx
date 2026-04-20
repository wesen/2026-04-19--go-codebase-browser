import { greet } from './greeter.js';

interface CardProps {
  name: string;
}

export default function Card({ name }: CardProps) {
  return <div className="card">{greet(name)}</div>;
}

export function Footer() {
  return <footer />;
}
