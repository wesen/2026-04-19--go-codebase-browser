import initSqlJs from 'sql.js';
import type initSqlJsTypes from 'sql.js';
import { queryOne } from './sqlRows';

type Database = initSqlJsTypes.Database;
type SqlJsStatic = initSqlJsTypes.SqlJsStatic;

export interface StaticManifest {
  schemaVersion?: number;
  kind?: string;
  generatedAt?: string;
  db?: {
    path?: string;
    sizeBytes?: number;
    schemaVersion?: number;
  };
  features?: Record<string, boolean>;
  runtime?: Record<string, unknown>;
}

let sqlJsPromise: Promise<SqlJsStatic> | null = null;
let manifestPromise: Promise<StaticManifest> | null = null;
let dbPromise: Promise<Database> | null = null;

export async function getSqlJs(): Promise<SqlJsStatic> {
  if (!sqlJsPromise) {
    sqlJsPromise = initSqlJs({
      locateFile: (file) => (file === 'sql-wasm.wasm' ? 'sql-wasm.wasm' : file),
    });
  }
  return sqlJsPromise;
}

export async function getStaticManifest(): Promise<StaticManifest> {
  if (!manifestPromise) {
    manifestPromise = (async () => {
      const response = await fetch('manifest.json');
      if (!response.ok) {
        return { db: { path: 'db/codebase.db' } } satisfies StaticManifest;
      }
      return (await response.json()) as StaticManifest;
    })();
  }
  return manifestPromise;
}

export async function getStaticDb(): Promise<Database> {
  if (!dbPromise) {
    dbPromise = (async () => {
      const [SQL, manifest] = await Promise.all([getSqlJs(), getStaticManifest()]);
      const dbPath = manifest.db?.path ?? 'db/codebase.db';
      const response = await fetch(dbPath);
      if (!response.ok) {
        throw new Error(`failed to fetch SQLite DB ${dbPath}: ${response.status} ${response.statusText}`);
      }
      const bytes = new Uint8Array(await response.arrayBuffer());
      return new SQL.Database(bytes);
    })();
  }
  return dbPromise;
}

export async function smokeCountCommits(): Promise<number> {
  const db = await getStaticDb();
  const row = queryOne<{ count: number }>(db, 'SELECT COUNT(*) AS count FROM commits');
  return Number(row?.count ?? 0);
}

export function resetStaticDbForTests(): void {
  dbPromise = null;
  manifestPromise = null;
  sqlJsPromise = null;
}
