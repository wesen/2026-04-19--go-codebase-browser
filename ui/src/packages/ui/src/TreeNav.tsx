// React namespace provided by jsx: react-jsx
import { PARTS } from './parts';

export interface TreeItem {
  id: string;
  label: string;
  hint?: string;
  href?: string;
  onClick?: () => void;
  active?: boolean;
}

export interface TreeNavProps {
  items: TreeItem[];
}

export function TreeNav({ items }: TreeNavProps) {
  if (!items.length) {
    return <div data-part={PARTS.empty}>No items</div>;
  }
  return (
    <ul data-part={PARTS.treeNav}>
      {items.map((item) => (
        <li key={item.id}>
          <a
            data-part={PARTS.treeNode}
            data-state={item.active ? 'active' : undefined}
            href={item.href}
            onClick={(e) => {
              if (item.onClick) {
                e.preventDefault();
                item.onClick();
              }
            }}
          >
            {item.label}
            {item.hint && <span data-role="hint"> ({item.hint})</span>}
          </a>
        </li>
      ))}
    </ul>
  );
}
