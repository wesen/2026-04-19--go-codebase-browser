import type {
  BodyDiffResult,
  CommitDiff,
  CommitRow,
  ImpactResponse,
  SymbolHistoryEntry,
} from './historyApi';
import { SqlJsQueryProvider } from './sqlJsQueryProvider';

export interface CodebaseQueryProvider {
  listCommits(): Promise<CommitRow[]>;
  getCommit(ref: string): Promise<CommitRow>;
  resolveCommitRef(ref: string): Promise<string>;
  getSymbolHistory(symbolId: string): Promise<SymbolHistoryEntry[]>;
  getSymbolBodyDiff(from: string, to: string, symbolId: string): Promise<BodyDiffResult>;
  getCommitDiff(from: string, to: string): Promise<CommitDiff>;
  getImpact(options: {
    symbolId: string;
    direction: 'usedby' | 'uses';
    depth: number;
    commit?: string;
  }): Promise<ImpactResponse>;
}

let provider: CodebaseQueryProvider | null = null;

export function getQueryProvider(): CodebaseQueryProvider {
  if (!provider) provider = new SqlJsQueryProvider();
  return provider;
}

export function resetQueryProviderForTests(): void {
  provider = null;
}
