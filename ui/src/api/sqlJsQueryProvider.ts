import type { DocPage, ReviewDocMeta, SnippetRef } from './docApi';
import type {
  BodyDiffResult,
  CommitDiff,
  CommitRow,
  DiffStats,
  FileDiff,
  ImpactEdge,
  ImpactNode,
  ImpactResponse,
  SymbolDiff,
  SymbolHistoryEntry,
} from './historyApi';
import { QueryError } from './queryErrors';
import { getStaticDb } from './sqljs/sqlJsDb';
import { extractUtf8Range, queryAll, queryOne, sqlBlobToBytes, type SqlRow } from './sqljs/sqlRows';

type CommitRowSQL = SqlRow & {
  Hash: string;
  ShortHash: string;
  Message: string;
  AuthorName: string;
  AuthorEmail: string;
  AuthorTime: number;
  IndexedAt: number;
  Branch: string;
  Error: string;
};

type SymbolHistorySQL = SqlRow & {
  symbolId: string;
  name: string;
  kind: string;
  packageId: string;
  commitHash: string;
  shortHash: string;
  message: string;
  authorTime: number;
  bodyHash: string;
  startLine: number;
  endLine: number;
  signature: string;
  fileId: string;
};

type BodyMetaSQL = SqlRow & {
  symbolId: string;
  name: string;
  startOffset: number;
  endOffset: number;
  startLine: number;
  endLine: number;
  filePath: string;
  contentHash: string;
};

type ContentSQL = SqlRow & {
  content: Uint8Array;
};

type FileDiffSQL = SqlRow & FileDiff;

type SymbolDiffSQL = SqlRow & SymbolDiff;

type ImpactEdgeSQL = SqlRow & ImpactEdge;

type SymbolMetaSQL = SqlRow & {
  id: string;
  name: string;
  kind: string;
};

type ReviewDocMetaSQL = SqlRow & ReviewDocMeta;

type ReviewDocSQL = SqlRow & {
  slug: string;
  title: string;
  html: string;
  snippetsJson: string;
  errorsJson: string;
};

let provider: SqlJsQueryProvider | null = null;

export function getSqlJsProvider(): SqlJsQueryProvider {
  if (!provider) provider = new SqlJsQueryProvider();
  return provider;
}

export function resetSqlJsProviderForTests(): void {
  provider = null;
}

export class SqlJsQueryProvider {
  async listCommits(): Promise<CommitRow[]> {
    const db = await getStaticDb();
    return queryAll<CommitRowSQL>(db, `
      SELECT hash AS Hash,
             short_hash AS ShortHash,
             message AS Message,
             author_name AS AuthorName,
             author_email AS AuthorEmail,
             author_time AS AuthorTime,
             indexed_at AS IndexedAt,
             branch AS Branch,
             error AS Error
      FROM commits
      WHERE error = ''
      ORDER BY author_time DESC
    `).map((row) => ({ ...row }));
  }

  async resolveCommitRef(ref: string): Promise<string> {
    const commits = await this.listCommits();
    if (commits.length === 0) throw new QueryError('NOT_FOUND', 'no indexed commits in database');

    const ordered = [...commits].sort((a, b) => a.AuthorTime - b.AuthorTime);
    const newestIndex = ordered.length - 1;
    if (!ref || ref === 'HEAD') return ordered[newestIndex].Hash;

    const headOffset = ref.match(/^HEAD~(\d+)$/);
    if (headOffset) {
      const index = newestIndex - Number.parseInt(headOffset[1], 10);
      const commit = ordered[index];
      if (!commit) throw new QueryError('NOT_FOUND', `commit ref not found: ${ref}`);
      return commit.Hash;
    }

    const exact = ordered.find((commit) => commit.Hash === ref || commit.ShortHash === ref);
    if (exact) return exact.Hash;

    const prefixMatches = ordered.filter((commit) => commit.Hash.startsWith(ref));
    if (prefixMatches.length === 1) return prefixMatches[0].Hash;
    if (prefixMatches.length > 1) throw new QueryError('AMBIGUOUS_REF', `ambiguous commit ref: ${ref}`);

    throw new QueryError('NOT_FOUND', `commit ref not found: ${ref}`);
  }

  async getCommit(ref: string): Promise<CommitRow> {
    const hash = await this.resolveCommitRef(ref);
    const commits = await this.listCommits();
    const commit = commits.find((row) => row.Hash === hash);
    if (!commit) throw new QueryError('NOT_FOUND', `commit not found: ${ref}`);
    return commit;
  }

  async getSymbolHistory(symbolId: string): Promise<SymbolHistoryEntry[]> {
    const db = await getStaticDb();
    return queryAll<SymbolHistorySQL>(db, `
      SELECT symbol_id AS symbolId,
             name,
             kind,
             package_id AS packageId,
             commit_hash AS commitHash,
             short_hash AS shortHash,
             commit_message AS message,
             author_time AS authorTime,
             body_hash AS bodyHash,
             start_line AS startLine,
             end_line AS endLine,
             signature,
             file_id AS fileId
      FROM symbol_history
      WHERE symbol_id = ?
      ORDER BY author_time DESC
    `, [symbolId]).map((row) => ({
      commitHash: row.commitHash,
      shortHash: row.shortHash,
      message: row.message,
      authorTime: row.authorTime,
      bodyHash: row.bodyHash,
      startLine: row.startLine,
      endLine: row.endLine,
      signature: row.signature,
      kind: row.kind,
    }));
  }

  async getSymbolBodyDiff(from: string, to: string, symbolId: string): Promise<BodyDiffResult> {
    const oldHash = await this.resolveCommitRef(from);
    const newHash = await this.resolveCommitRef(to);
    const [oldMeta, newMeta] = await Promise.all([
      this.getBodyMeta(oldHash, symbolId),
      this.getBodyMeta(newHash, symbolId),
    ]);
    const [oldBytes, newBytes] = await Promise.all([
      this.getContentBytes(oldMeta.contentHash),
      this.getContentBytes(newMeta.contentHash),
    ]);
    const oldBody = extractUtf8Range(oldBytes, oldMeta.startOffset, oldMeta.endOffset);
    const newBody = extractUtf8Range(newBytes, newMeta.startOffset, newMeta.endOffset);

    return {
      symbolId,
      name: newMeta.name || oldMeta.name,
      oldCommit: oldHash,
      newCommit: newHash,
      oldBody,
      newBody,
      unifiedDiff: simpleUnifiedDiff(oldBody, newBody),
      oldRange: `lines ${oldMeta.startLine}-${oldMeta.endLine}`,
      newRange: `lines ${newMeta.startLine}-${newMeta.endLine}`,
    };
  }

  async getCommitDiff(from: string, to: string): Promise<CommitDiff> {
    const oldHash = await this.resolveCommitRef(from);
    const newHash = await this.resolveCommitRef(to);
    const db = await getStaticDb();
    const files = queryAll<FileDiffSQL>(db, fileDiffSQL, [oldHash, newHash, newHash, oldHash, oldHash, newHash]);
    const symbols = queryAll<SymbolDiffSQL>(db, symbolDiffSQL, [oldHash, newHash, newHash, oldHash, oldHash, newHash]);
    return {
      OldHash: oldHash,
      NewHash: newHash,
      Files: files.map((row) => ({ ...row })),
      Symbols: symbols.map((row) => ({ ...row })),
      Stats: diffStats(files, symbols),
    };
  }

  async getImpact(options: {
    symbolId: string;
    direction: 'usedby' | 'uses';
    depth: number;
    commit?: string;
  }): Promise<ImpactResponse> {
    const commit = await this.resolveCommitRef(options.commit ?? 'HEAD');
    const direction = options.direction;
    const maxDepth = Math.max(1, Math.min(options.depth, 5));
    const visited = new Set<string>([options.symbolId]);
    const queue: Array<{ symbolId: string; depth: number }> = [{ symbolId: options.symbolId, depth: 0 }];
    const nodeByID = new Map<string, ImpactNode>();

    while (queue.length > 0) {
      const item = queue.shift();
      if (!item || item.depth >= maxDepth) continue;
      const edges = direction === 'uses'
        ? await this.getRefsFrom(item.symbolId, commit)
        : await this.getRefsTo(item.symbolId, commit);
      for (const edge of edges) {
        const nextID = direction === 'uses' ? edge.toSymbolId : edge.fromSymbolId;
        const nextDepth = item.depth + 1;
        let node = nodeByID.get(nextID);
        if (!node) {
          const meta = await this.getSymbolMeta(nextID, commit);
          node = {
            symbolId: nextID,
            name: meta?.name ?? fallbackName(nextID),
            kind: meta?.kind ?? 'external',
            depth: nextDepth,
            edges: [],
            compatibility: 'unknown',
            local: !!meta,
          };
          nodeByID.set(nextID, node);
        }
        node.edges.push(edge);
        if (!visited.has(nextID)) {
          visited.add(nextID);
          queue.push({ symbolId: nextID, depth: nextDepth });
        }
      }
    }

    return {
      root: options.symbolId,
      direction,
      depth: maxDepth,
      commit,
      nodes: [...nodeByID.values()],
    };
  }

  async listReviewDocs(): Promise<ReviewDocMeta[]> {
    const db = await getStaticDb();
    return queryAll<ReviewDocMetaSQL>(db, `
      SELECT slug, title
      FROM static_review_rendered_docs
      ORDER BY slug
    `).map((row) => ({ slug: row.slug, title: row.title }));
  }

  async getReviewDoc(slug: string): Promise<DocPage> {
    const db = await getStaticDb();
    const row = queryOne<ReviewDocSQL>(db, `
      SELECT slug,
             title,
             html,
             snippets_json AS snippetsJson,
             errors_json AS errorsJson
      FROM static_review_rendered_docs
      WHERE slug = ?
    `, [slug]);
    if (!row) throw new QueryError('NOT_FOUND', `review doc not found: ${slug}`);
    return {
      slug: row.slug,
      title: row.title,
      html: row.html,
      snippets: parseJSON<SnippetRef[]>(row.snippetsJson, []),
      errors: parseJSON<string[]>(row.errorsJson, []),
    };
  }

  private async getBodyMeta(commitHash: string, symbolId: string): Promise<BodyMetaSQL> {
    const db = await getStaticDb();
    const row = queryOne<BodyMetaSQL>(db, `
      SELECT s.id AS symbolId,
             s.name AS name,
             s.start_offset AS startOffset,
             s.end_offset AS endOffset,
             s.start_line AS startLine,
             s.end_line AS endLine,
             f.path AS filePath,
             COALESCE(NULLIF(f.content_hash, ''), f.sha256) AS contentHash
      FROM snapshot_symbols s
      JOIN snapshot_files f
        ON f.commit_hash = s.commit_hash
       AND f.id = s.file_id
      WHERE s.commit_hash = ? AND s.id = ?
    `, [commitHash, symbolId]);
    if (!row) throw new QueryError('NOT_FOUND', `symbol ${symbolId} not found at ${commitHash.slice(0, 7)}`);
    return row;
  }

  private async getContentBytes(contentHash: string): Promise<Uint8Array> {
    const db = await getStaticDb();
    const row = queryOne<ContentSQL>(db, `
      SELECT content
      FROM file_contents
      WHERE content_hash = ?
    `, [contentHash]);
    if (!row) throw new QueryError('NOT_FOUND', `file content not found: ${contentHash}`);
    return sqlBlobToBytes(row.content);
  }

  private async getRefsFrom(symbolId: string, commit: string): Promise<ImpactEdge[]> {
    const db = await getStaticDb();
    return queryAll<ImpactEdgeSQL>(db, `
      SELECT from_symbol_id AS fromSymbolId,
             to_symbol_id AS toSymbolId,
             kind,
             file_id AS fileId
      FROM snapshot_refs
      WHERE commit_hash = ? AND from_symbol_id = ?
      ORDER BY to_symbol_id, kind
    `, [commit, symbolId]).map((row) => ({ ...row }));
  }

  private async getRefsTo(symbolId: string, commit: string): Promise<ImpactEdge[]> {
    const db = await getStaticDb();
    return queryAll<ImpactEdgeSQL>(db, `
      SELECT from_symbol_id AS fromSymbolId,
             to_symbol_id AS toSymbolId,
             kind,
             file_id AS fileId
      FROM snapshot_refs
      WHERE commit_hash = ? AND to_symbol_id = ?
      ORDER BY from_symbol_id, kind
    `, [commit, symbolId]).map((row) => ({ ...row }));
  }

  private async getSymbolMeta(symbolId: string, commit: string): Promise<SymbolMetaSQL | null> {
    const db = await getStaticDb();
    return queryOne<SymbolMetaSQL>(db, `
      SELECT id, name, kind
      FROM snapshot_symbols
      WHERE commit_hash = ? AND id = ?
    `, [commit, symbolId]);
  }
}

const fileDiffSQL = `
  SELECT b.id AS FileID,
         b.path AS Path,
         'added' AS ChangeType,
         '' AS OldSHA256,
         b.sha256 AS NewSHA256
  FROM snapshot_files b
  LEFT JOIN snapshot_files a
    ON a.commit_hash = ? AND a.id = b.id
  WHERE b.commit_hash = ? AND a.id IS NULL

  UNION ALL

  SELECT a.id AS FileID,
         a.path AS Path,
         'removed' AS ChangeType,
         a.sha256 AS OldSHA256,
         '' AS NewSHA256
  FROM snapshot_files a
  LEFT JOIN snapshot_files b
    ON b.commit_hash = ? AND b.id = a.id
  WHERE a.commit_hash = ? AND b.id IS NULL

  UNION ALL

  SELECT b.id AS FileID,
         b.path AS Path,
         'modified' AS ChangeType,
         a.sha256 AS OldSHA256,
         b.sha256 AS NewSHA256
  FROM snapshot_files a
  JOIN snapshot_files b
    ON b.id = a.id
  WHERE a.commit_hash = ?
    AND b.commit_hash = ?
    AND a.sha256 != b.sha256
  ORDER BY Path
`;

const symbolDiffSQL = `
  SELECT b.id AS SymbolID,
         b.name AS Name,
         b.kind AS Kind,
         b.package_id AS PackageID,
         'added' AS ChangeType,
         0 AS OldStartLine,
         0 AS OldEndLine,
         b.start_line AS NewStartLine,
         b.end_line AS NewEndLine,
         '' AS OldSignature,
         b.signature AS NewSignature,
         '' AS OldBodyHash,
         b.body_hash AS NewBodyHash
  FROM snapshot_symbols b
  LEFT JOIN snapshot_symbols a
    ON a.commit_hash = ? AND a.id = b.id
  WHERE b.commit_hash = ? AND a.id IS NULL

  UNION ALL

  SELECT a.id AS SymbolID,
         a.name AS Name,
         a.kind AS Kind,
         a.package_id AS PackageID,
         'removed' AS ChangeType,
         a.start_line AS OldStartLine,
         a.end_line AS OldEndLine,
         0 AS NewStartLine,
         0 AS NewEndLine,
         a.signature AS OldSignature,
         '' AS NewSignature,
         a.body_hash AS OldBodyHash,
         '' AS NewBodyHash
  FROM snapshot_symbols a
  LEFT JOIN snapshot_symbols b
    ON b.commit_hash = ? AND b.id = a.id
  WHERE a.commit_hash = ? AND b.id IS NULL

  UNION ALL

  SELECT b.id AS SymbolID,
         b.name AS Name,
         b.kind AS Kind,
         b.package_id AS PackageID,
         CASE
           WHEN a.body_hash != b.body_hash AND a.body_hash != '' AND b.body_hash != '' THEN 'modified'
           WHEN a.signature != b.signature THEN 'signature-changed'
           WHEN a.start_line != b.start_line OR a.end_line != b.end_line THEN 'moved'
           ELSE 'unchanged'
         END AS ChangeType,
         a.start_line AS OldStartLine,
         a.end_line AS OldEndLine,
         b.start_line AS NewStartLine,
         b.end_line AS NewEndLine,
         a.signature AS OldSignature,
         b.signature AS NewSignature,
         a.body_hash AS OldBodyHash,
         b.body_hash AS NewBodyHash
  FROM snapshot_symbols a
  JOIN snapshot_symbols b
    ON b.id = a.id
  WHERE a.commit_hash = ?
    AND b.commit_hash = ?
    AND (
      a.body_hash != b.body_hash
      OR a.signature != b.signature
      OR a.start_line != b.start_line
      OR a.end_line != b.end_line
    )
  ORDER BY Name
`;

function diffStats(files: FileDiff[], symbols: SymbolDiff[]): DiffStats {
  const stats: DiffStats = {
    FilesAdded: 0,
    FilesRemoved: 0,
    FilesModified: 0,
    SymbolsAdded: 0,
    SymbolsRemoved: 0,
    SymbolsModified: 0,
    SymbolsMoved: 0,
    SymbolsUnchanged: 0,
  };
  for (const file of files) {
    if (file.ChangeType === 'added') stats.FilesAdded++;
    else if (file.ChangeType === 'removed') stats.FilesRemoved++;
    else if (file.ChangeType === 'modified') stats.FilesModified++;
  }
  for (const symbol of symbols) {
    if (symbol.ChangeType === 'added') stats.SymbolsAdded++;
    else if (symbol.ChangeType === 'removed') stats.SymbolsRemoved++;
    else if (symbol.ChangeType === 'modified') stats.SymbolsModified++;
    else if (symbol.ChangeType === 'moved') stats.SymbolsMoved++;
    else if (symbol.ChangeType === 'unchanged') stats.SymbolsUnchanged++;
  }
  return stats;
}

function fallbackName(symbolId: string): string {
  const trimmed = symbolId.startsWith('sym:') ? symbolId.slice(4) : symbolId;
  const lastDot = trimmed.lastIndexOf('.');
  return lastDot >= 0 ? trimmed.slice(lastDot + 1) : trimmed;
}

function parseJSON<T>(raw: string, fallback: T): T {
  try {
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function simpleUnifiedDiff(oldText: string, newText: string): string {
  const oldLines = splitLines(oldText);
  const newLines = splitLines(newText);
  let prefix = 0;
  while (prefix < oldLines.length && prefix < newLines.length && oldLines[prefix] === newLines[prefix]) {
    prefix++;
  }

  let suffix = 0;
  while (
    suffix < oldLines.length - prefix &&
    suffix < newLines.length - prefix &&
    oldLines[oldLines.length - 1 - suffix] === newLines[newLines.length - 1 - suffix]
  ) {
    suffix++;
  }

  const out: string[] = [];
  for (let i = 0; i < prefix; i++) out.push(`  ${oldLines[i]}`);
  for (let i = prefix; i < oldLines.length - suffix; i++) out.push(`- ${oldLines[i]}`);
  for (let i = prefix; i < newLines.length - suffix; i++) out.push(`+ ${newLines[i]}`);
  for (let i = oldLines.length - suffix; i < oldLines.length; i++) out.push(`  ${oldLines[i]}`);
  return out.join('\n') + (out.length > 0 ? '\n' : '');
}

function splitLines(text: string): string[] {
  const lines = text.split('\n');
  if (lines.length > 0 && lines[lines.length - 1] === '') lines.pop();
  return lines;
}
