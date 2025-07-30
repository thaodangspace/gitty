export interface Repository {
  id: string;
  name: string;
  path: string;
  url?: string;
  description?: string;
  created_at: string;
  updated_at: string;
  is_local: boolean;
  current_branch?: string;
}

export interface RepositoryStatus {
  repository_id: string;
  branch: string;
  is_clean: boolean;
  ahead: number;
  behind: number;
  staged: FileChange[];
  modified: FileChange[];
  untracked: string[];
  conflicts: string[];
}

export interface FileChange {
  path: string;
  status: string;
  type: string;
}

export interface Branch {
  name: string;
  is_current: boolean;
  is_remote: boolean;
  upstream?: string;
  last_commit?: Commit;
}

export interface Commit {
  hash: string;
  message: string;
  author: Author;
  date: string;
  parent_hash?: string;
}

export interface Author {
  name: string;
  email: string;
}

export interface FileInfo {
  path: string;
  name: string;
  is_directory: boolean;
  size: number;
  mod_time: string;
  mode: string;
}

export interface CreateRepositoryRequest {
  name: string;
  path?: string;
  url?: string;
  description?: string;
  is_local: boolean;
}

export interface CommitRequest {
  message: string;
  files: string[];
  author?: Author;
}

export interface DirectoryEntry {
  name: string;
  path: string;
  is_directory: boolean;
  is_hidden: boolean;
  size?: number;
  mod_time?: string;
  permissions?: string;
  is_git_repo: boolean;
}

export interface DirectoryListing {
  current_path: string;
  parent_path?: string;
  entries: DirectoryEntry[];
  can_go_up: boolean;
}

export interface ApiError {
  message: string;
  status: number;
}

export interface CommitDetail {
  hash: string;
  message: string;
  author: Author;
  date: string;
  parent_hash?: string;
  changes: FileDiff[];
  stats: DiffStats;
}

export interface FileDiff {
  path: string;
  change_type: string;
  additions: number;
  deletions: number;
  patch: string;
}

export interface DiffStats {
  additions: number;
  deletions: number;
  files_changed: number;
}