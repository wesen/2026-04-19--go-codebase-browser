// Godoc / inline-comment annotations. Detects the handful of well-known
// prefixes inside `// …` and `/* … */` comments and returns a split so the
// renderer can tag each span with data-annotation.

export type Annotation = 'deprecated' | 'bug' | 'todo' | 'fixme' | 'note' | 'hack' | 'xxx';

export interface AnnotationSpan {
  annotation?: Annotation;
  text: string;
}

// Godoc conventions: https://go.dev/blog/godoc
// These are the prefixes I see in the wild and want visually distinct.
const PATTERNS: Array<{ annotation: Annotation; re: RegExp }> = [
  { annotation: 'deprecated', re: /\bDeprecated:\s.*/ },
  { annotation: 'bug', re: /\bBUG(?:\([^)]*\))?:\s.*/ },
  { annotation: 'todo', re: /\bTODO(?:\([^)]*\))?:\s.*/ },
  { annotation: 'fixme', re: /\bFIXME(?:\([^)]*\))?:\s.*/ },
  { annotation: 'note', re: /\bNOTE(?:\([^)]*\))?:\s.*/ },
  { annotation: 'hack', re: /\bHACK(?:\([^)]*\))?:\s.*/ },
  { annotation: 'xxx', re: /\bXXX:\s.*/ },
];

/**
 * annotateComment scans a comment token's text and returns a list of spans,
 * each either plain (no annotation) or carrying one of the known annotation
 * markers. The concatenation of spans reproduces the original input.
 */
export function annotateComment(text: string): AnnotationSpan[] {
  for (const { annotation, re } of PATTERNS) {
    const m = re.exec(text);
    if (!m) continue;
    const before = text.slice(0, m.index);
    const matched = m[0];
    const after = text.slice(m.index + matched.length);
    const out: AnnotationSpan[] = [];
    if (before) out.push({ text: before });
    out.push({ annotation, text: matched });
    if (after) out.push({ text: after });
    return out;
  }
  return [{ text }];
}

/**
 * detectLeadingAnnotation returns an annotation kind if the given doc text
 * starts with one of the known prefixes. Used to stamp a badge onto
 * SymbolCard without re-rendering the entire doc block.
 */
export function detectLeadingAnnotation(doc: string): Annotation | undefined {
  const firstLine = (doc.split('\n')[0] ?? '').trim();
  for (const { annotation, re } of PATTERNS) {
    if (re.test(firstLine)) return annotation;
  }
  return undefined;
}
