import { useState, useEffect } from 'react';
import { X, Loader2 } from 'lucide-react';
import { Button } from '../ui/button';
import { apiClient } from '../../lib/api-client';
import { PatchDiff } from '@pierre/diffs/react';

interface DiffViewerProps {
    repositoryId: string;
    filePath: string;
    fileName: string;
    onClose: () => void;
}

export default function DiffViewer({ repositoryId, filePath, fileName, onClose }: DiffViewerProps) {
    const [diffText, setDiffText] = useState<string>('');
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchDiff = async () => {
            try {
                setIsLoading(true);
                setError(null);
                const diffContent = await apiClient.getFileDiff(repositoryId, filePath);
                setDiffText(diffContent);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to load diff');
            } finally {
                setIsLoading(false);
            }
        };

        fetchDiff();
    }, [repositoryId, filePath]);

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
                    ) : !diffText ? (
                        <div className="flex items-center justify-center h-48">
                            <p className="text-muted-foreground dark:text-gray-400">No changes to display</p>
                        </div>
                    ) : (
                        <div className="diff-view-container p-4">
                            <PatchDiff
                                patch={diffText}
                                options={{
                                    diffStyle: 'unified',
                                    theme: {
                                        dark: 'github-dark',
                                        light: 'github-light',
                                    },
                                }}
                            />
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
