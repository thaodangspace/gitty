import { useQuery } from '@tanstack/react-query';
import axios from 'axios';
import type { TokenizedDiff, TokenizedCommitDiff } from '../types/diff';
import { API_BASE } from '../config';

// ─── Fetch tokenized diff for a single file (working tree or staged) ───

interface UseFileDiffOptions {
  repoPath: string;
  filePath: string;
  staged?: boolean;
  enabled?: boolean;
}

export function useFileDiff({
  repoPath,
  filePath,
  staged = false,
  enabled = true,
}: UseFileDiffOptions) {
  return useQuery<TokenizedDiff>({
    queryKey: ['fileDiff', repoPath, filePath, staged],
    queryFn: async () => {
      const { data } = await axios.get(`${API_BASE}/api/diff/file`, {
        params: { repo: repoPath, path: filePath, staged: staged.toString() },
      });
      return data;
    },
    enabled: enabled && !!repoPath && !!filePath,
    staleTime: 5_000, // 5s — diffs change frequently during dev
    gcTime: 30_000,
  });
}

// ─── Fetch tokenized diff for an entire commit ───

interface UseCommitDiffOptions {
  repoPath: string;
  commitHash: string;
  enabled?: boolean;
}

export function useCommitDiff({
  repoPath,
  commitHash,
  enabled = true,
}: UseCommitDiffOptions) {
  return useQuery<TokenizedCommitDiff>({
    queryKey: ['commitDiff', repoPath, commitHash],
    queryFn: async () => {
      const { data } = await axios.get(`${API_BASE}/api/diff/commit`, {
        params: { repo: repoPath, hash: commitHash },
      });
      return data;
    },
    enabled: enabled && !!repoPath && !!commitHash,
    staleTime: 60_000, // Commits are immutable, cache longer
    gcTime: 5 * 60_000,
  });
}
