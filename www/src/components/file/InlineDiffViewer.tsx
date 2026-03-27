import { useState, useEffect, useCallback } from 'react';
import { Loader2 } from 'lucide-react';
import { apiClient } from '../../lib/api-client';
import TokenizedDiffRenderer from './TokenizedDiffRenderer';
import type { TokenizedDiff } from '../../types/api';

interface InlineDiffViewerProps {
    repositoryId: string;
    filePath: string;
    commitHash?: string;
}

export default function InlineDiffViewer({ repositoryId, filePath, commitHash }: InlineDiffViewerProps) {
    const [diff, setDiff] = useState<TokenizedDiff | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const loadDiff = useCallback(async () => {
        try {
            setIsLoading(true);
            setError(null);
            let diffContent: TokenizedDiff;
            if (commitHash) {
                diffContent = await apiClient.getCommitFileDiff(repositoryId, commitHash, filePath);
            } else {
                diffContent = await apiClient.getTokenizedFileDiff(repositoryId, filePath);
            }
            setDiff(diffContent);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load diff');
        } finally {
            setIsLoading(false);
        }
    }, [repositoryId, filePath, commitHash]);

    useEffect(() => {
        loadDiff();
    }, [loadDiff]);

    if (isLoading) {
        return (
            <div className="flex items-center justify-center h-32">
                <div className="flex items-center gap-2 text-muted-foreground">
                    <Loader2 className="h-5 w-5 animate-spin" />
                    Loading diff...
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-4 text-center">
                <p className="text-red-600">Error loading diff</p>
                <p className="text-sm text-muted-foreground">{error}</p>
            </div>
        );
    }

    if (!diff) {
        return (
            <div className="flex items-center justify-center h-32">
                <p className="text-muted-foreground">No changes to display</p>
            </div>
        );
    }

    return (
        <div className="p-4 overflow-auto">
            <TokenizedDiffRenderer diff={diff} />
        </div>
    );
}
