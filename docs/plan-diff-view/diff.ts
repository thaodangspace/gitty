// ─── Tokenized Diff Types ───
// Matches the Go models exactly. The server sends these, RN renders them.

export interface Token {
  text: string;
  color: string;
}

export interface DiffLine {
  type: 'added' | 'deleted' | 'context';
  tokens: Token[];
  oldNum?: number;
  newNum?: number;
}

export interface DiffHunk {
  header: string;
  lines: DiffLine[];
}

export interface TokenizedDiff {
  filename: string;
  hunks: DiffHunk[];
  additions: number;
  deletions: number;
}

export interface TokenizedFileDiff {
  path: string;
  changeType: 'added' | 'modified' | 'deleted';
  diff: TokenizedDiff;
}

export interface TokenizedCommitDiff {
  hash: string;
  message: string;
  author: {
    name: string;
    email: string;
  };
  date: string;
  files: TokenizedFileDiff[];
  stats: {
    additions: number;
    deletions: number;
  };
}
