// React namespace provided by jsx: react-jsx
import { PARTS } from './parts';

export interface SearchBoxProps {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
}

export function SearchBox({ value, onChange, placeholder }: SearchBoxProps) {
  return (
    <input
      data-part={PARTS.searchBox}
      type="search"
      value={value}
      placeholder={placeholder ?? 'Search symbols…'}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}
