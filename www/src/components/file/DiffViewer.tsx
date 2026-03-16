import { useState, useEffect, useRef, useCallback } from 'react';
import { X, Loader2, ChevronDown } from 'lucide-react';
import { Button } from '../ui/button';
import { apiClient } from '../../lib/api-client';
import TokenizedDiffRenderer from './TokenizedDiffRenderer';
import type { TokenizedDiff } from '../../types/api';

interface DiffViewerProps {
    repositoryId: string;
    filePath: string;
    fileName: string;
    onClose: () => void;
}

export default function DiffViewer({ repositoryId, filePath, fileName, onClose }: DiffViewerProps) {
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
            const nextDiff = await apiClient.getTokenizedFileDiff(repositoryId, filePath, false, diff.next_cursor);
            setDiff(prev => {
                if (!prev) return nextDiff;
                return {
                    ...prev,
                    hunks: [...prev.hunks, ...nextDiff.hunks],
                    additions: prev.additions + nextDiff.additions, // note: backend might return total or page additions
                    deletions: prev.deletions + nextDiff.deletions,
                    has_more: nextDiff.has_more,
                    next_cursor: nextDiff.next_cursor,
                    total_hunks: nextDiff.total_hunks,
                };
            });
        } catch (err) {
            console.error("Failed to load more hunks", err);
        } finally {
            setIsLoadingMore(false);
        }
    }, [diff?.has_more, diff?.next_cursor, isLoadingMore, repositoryId, filePath]);

    useEffect(() => {
        const fetchInitialDiff = async () => {
            try {
                setIsLoading(true);
                setError(null);
                const diffContent = await apiClient.getTokenizedFileDiff(repositoryId, filePath);
                setDiff(diffContent);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to load diff');
            } finally {
                setIsLoading(false);
            }
        };

        fetchInitialDiff();
    }, [repositoryId, filePath]);

    useEffect(() => {
        const option = {
            root: null,
            rootMargin: "20px",
            threshold: 1.0
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

    return (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
            <div className="bg-white rounded-lg shadow-xl max-w-6xl w-full max-h-[90vh] flex flex-col dark:bg-gray-900">
                <div className="flex items-center justify-between p-4 border-b dark:border-gray-700">
                    <div>
                        <h2 className="text-lg font-semibold dark:text-white">Changes in {fileName}</h2>
                        <p className="text-sm text-muted-foreground dark:text-gray-400">{filePath}</p>
                    </div>
                    <Button variant="ghost" size="sm" onClick={onClose} className="h-8 w-8 p-0 dark:hover:bg-gray-800">
                        <X className="h-4 w-4 dark:text-gray-300" />
                    </Button>
                </div>

                <div className="flex-1 overflow-auto">
                    {isLoading ? (
                        <div className="flex items-center justify-center h-48">
                            <div className="flex items-center gap-2 text-muted-foreground dark:text-gray-400">
                                <Loader2 className="h-5 w-5 animate-spin" />
                                Loading diff...
                            </div>
                        </div>
                    ) : error ? (
                        <div className="flex items-center justify-center h-48">
                            <div className="text-center">
                                <p className="text-red-600 mb-2 dark:text-red-400">Error loading diff</p>
                                <p className="text-sm text-muted-foreground dark:text-gray-400">{error}</p>
                            </div>
                        </div>
                    ) : !diff ? (
                        <div className="flex items-center justify-center h-48">
                            <p className="text-muted-foreground dark:text-gray-400">No changes to display</p>
                        </div>
                    ) : (
                        <div className="diff-view-container p-4">
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
                                        {isLoadingMore ? 'Loading...' : `Load more hunks (${diff.hunks.length} of ${diff.total_hunks || '??'})`}
                                    </Button>
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
