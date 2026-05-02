import initSqlJs from 'sql.js';
import { describe, expect, it } from 'vitest';

import { extractUtf8Range, queryAll, queryOne, sqlBlobToBytes, sqlBlobToText } from './sqlRows';

describe('sqlRows', () => {
  it('reads all and single rows from a sql.js database', async () => {
    const SQL = await initSqlJs();
    const db = new SQL.Database();
    try {
      db.run('CREATE TABLE things (id INTEGER PRIMARY KEY, name TEXT NOT NULL)');
      db.run('INSERT INTO things (name) VALUES (?), (?)', ['alpha', 'beta']);

      expect(queryAll(db, 'SELECT id, name FROM things ORDER BY id')).toEqual([
        { id: 1, name: 'alpha' },
        { id: 2, name: 'beta' },
      ]);
      expect(queryOne(db, 'SELECT id, name FROM things WHERE name = ?', ['beta'])).toEqual({
        id: 2,
        name: 'beta',
      });
      expect(queryOne(db, 'SELECT id, name FROM things WHERE name = ?', ['missing'])).toBeNull();
    } finally {
      db.close();
    }
  });

  it('normalizes SQLite BLOB values into bytes and text', () => {
    expect(Array.from(sqlBlobToBytes(new Uint8Array([0x68, 0x69])))).toEqual([0x68, 0x69]);
    expect(Array.from(sqlBlobToBytes([0x68, 0x69]))).toEqual([0x68, 0x69]);
    expect(sqlBlobToText('hé')).toBe('hé');
    expect(sqlBlobToText(undefined)).toBe('');
  });

  it('slices UTF-8 text by byte offsets before decoding', () => {
    const bytes = new TextEncoder().encode('a🙂b');

    expect(extractUtf8Range(bytes, 0, 1)).toBe('a');
    expect(extractUtf8Range(bytes, 1, 5)).toBe('🙂');
    expect(extractUtf8Range(bytes, 5, 6)).toBe('b');
  });

  it('rejects invalid byte ranges', () => {
    const bytes = new TextEncoder().encode('abc');

    expect(() => extractUtf8Range(bytes, -1, 1)).toThrow(/invalid byte range/);
    expect(() => extractUtf8Range(bytes, 2, 1)).toThrow(/invalid byte range/);
    expect(() => extractUtf8Range(bytes, 0, 4)).toThrow(/invalid byte range/);
  });
});
