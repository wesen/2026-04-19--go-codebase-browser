import type {
  BodyDiffResult,
  CommitDiff,
  CommitRow,
  DiffStats,
  ImpactResponse,
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
    return {
      OldHash: oldHash,
      NewHash: newHash,
      Files: [],
      Symbols: [],
      Stats: emptyDiffStats(),
    };
  }

  async getImpact(options: {
    symbolId: string;
    direction: 'usedby' | 'uses';
    depth: number;
    commit?: string;
  }): Promise<ImpactResponse> {
    const commit = await this.resolveCommitRef(options.commit ?? 'HEAD');
    return {
      root: options.symbolId,
      direction: options.direction,
      depth: options.depth,
      commit,
      nodes: [],
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
}

function emptyDiffStats(): DiffStats {
  return {
    FilesAdded: 0,
    FilesRemoved: 0,
    FilesModified: 0,
    SymbolsAdded: 0,
    SymbolsRemoved: 0,
    SymbolsModified: 0,
    SymbolsMoved: 0,
    SymbolsUnchanged: 0,
  };
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
