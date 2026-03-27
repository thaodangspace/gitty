import { useState, useEffect, useRef, useCallback } from 'react';
import { Loader2, ChevronDown, AlertCircle } from 'lucide-react';
import { Button } from '../ui/button';
import { apiClient } from '../../lib/api-client';
import TokenizedDiffRenderer from './TokenizedDiffRenderer';
import type { TokenizedDiff } from '../../types/api';

interface InlineDiffViewerProps {
  repositoryId: string;
  filePath: string;
  commitHash: string;
}

export default function InlineDiffViewer({
  repositoryId,
  filePath,
  commitHash,
}: InlineDiffViewerProps) {
  const [diff, setDiff] = useState<TokenizedDiff | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const loadMoreRef = useRef<HTMLDivElement | null>(null);

  const loadMore = useCallback(async () => {
    if (!diff?.has_more || diff?.next_cursor === undefined || isLoadingMore) return;

    try {
      setIsLoadingMore(true);
      const nextDiff = await apiClient.getCommitFileDiff(
        repositoryId,
        commitHash,
        filePath,
        diff.next_cursor
      );
      setDiff(prev => {
        if (!prev) return nextDiff;
        return {
          ...prev,
          hunks: [...prev.hunks, ...nextDiff.hunks],
          additions: prev.additions + nextDiff.additions,
          deletions: prev.deletions + nextDiff.deletions,
          has_more: nextDiff.has_more,
          next_cursor: nextDiff.next_cursor,
          total_hunks: nextDiff.total_hunks,
        };
      });
    } catch (err) {
      console.error('Failed to load more hunks', err);
    } finally {
      setIsLoadingMore(false);
    }
  }, [diff?.has_more, diff?.next_cursor, isLoadingMore, repositoryId, filePath, commitHash]);

  useEffect(() => {
    const fetchInitialDiff = async () => {
      try {
        setIsLoading(true);
        setError(null);
        const diffContent = await apiClient.getCommitFileDiff(
          repositoryId,
          commitHash,
          filePath
        );
        setDiff(diffContent);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load diff');
      } finally {
        setIsLoading(false);
      }
    };

    fetchInitialDiff();
  }, [repositoryId, filePath, commitHash]);

  useEffect(() => {
    const option = {
      root: null,
      rootMargin: '20px',
      threshold: 1.0,
    };
    const observer = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting && diff?.has_more) {
        loadMore();
      }
    }, option);
    if (loadMoreRef.current) observer.observe(loadMoreRef.current);
    observerRef.current = observer;

    return () => {
      if (observerRef.current) observerRef.current.disconnect();
    };
  }, [loadMore, diff?.has_more]);

  const handleRetry = () => {
    setError(null);
    setIsLoading(true);
    apiClient
      .getCommitFileDiff(repositoryId, commitHash, filePath)
      .then((diffContent) => {
        setDiff(diffContent);
        setIsLoading(false);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Failed to load diff');
        setIsLoading(false);
      });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8 bg-gray-950">
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading diff...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-8 bg-gray-950">
        <div className="flex items-center gap-2 text-red-400 mb-3">
          <AlertCircle className="h-4 w-4" />
          <span>Error loading diff</span>
        </div>
        <p className="text-sm text-muted-foreground mb-4">{error}</p>
        <Button variant="outline" size="sm" onClick={handleRetry}>
          Retry
        </Button>
      </div>
    );
  }

  if (!diff) {
    return (
      <div className="flex items-center justify-center py-8 bg-gray-950">
        <p className="text-muted-foreground">No diff available</p>
      </div>
    );
  }

  return (
    <div className="bg-gray-950">
      <div className="diff-view-container py-4">
        <TokenizedDiffRenderer diff={diff} />

        {diff.has_more && (
          <div ref={loadMoreRef} className="flex justify-center mt-4 pb-4">
            <Button
              variant="outline"
              onClick={loadMore}
              disabled={isLoadingMore}
              className="gap-2"
            >
              {isLoadingMore ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
              {isLoadingMore
                ? 'Loading...'
                : `Load more hunks (${diff.hunks.length} of ${diff.total_hunks || '??'})`}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
