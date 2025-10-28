import { useState, useEffect } from 'react';
import { X, Loader2, ChevronDown } from 'lucide-react';
import { Button } from '../ui/button';
import { apiClient } from '../../lib/api-client';

interface DiffViewerProps {
    repositoryId: string;
    filePath: string;
    fileName: string;
    onClose: () => void;
}

interface DiffLine {
    type: 'add' | 'remove' | 'context' | 'header';
    content: string;
    oldLineNumber?: number;
    newLineNumber?: number;
}

interface CollapsibleSection {
    id: string;
    startIndex: number;
    endIndex: number;
    totalLines: number;
    expandFromEnd: boolean; // true = expand from end (backwards), false = expand from start
}

export default function DiffViewer({ repositoryId, filePath, fileName, onClose }: DiffViewerProps) {
    const [diff, setDiff] = useState<string>('');
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    // Track how many lines are revealed in each section (sectionId -> number of revealed lines)
    const [revealedLines, setRevealedLines] = useState<Map<string, number>>(new Map());

    useEffect(() => {
        const fetchDiff = async () => {
            try {
                setIsLoading(true);
                setError(null);
                const diffContent = await apiClient.getFileDiff(repositoryId, filePath);
                setDiff(diffContent);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to load diff');
            } finally {
                setIsLoading(false);
            }
        };

        fetchDiff();
    }, [repositoryId, filePath]);

    const parseDiff = (diffText: string): DiffLine[] => {
        const lines = diffText.split('\n');
        const parsedLines: DiffLine[] = [];
        let oldLineNumber = 1;
        let newLineNumber = 1;

        for (const line of lines) {
            if (line.startsWith('@@')) {
                // Hunk header - extract line numbers
                const match = line.match(/@@ -(\d+),?\d* \+(\d+),?\d* @@/);
                if (match) {
                    oldLineNumber = parseInt(match[1]);
                    newLineNumber = parseInt(match[2]);
                }
                parsedLines.push({
                    type: 'header',
                    content: line,
                });
            } else if (line.startsWith('+')) {
                parsedLines.push({
                    type: 'add',
                    content: line.substring(1),
                    newLineNumber: newLineNumber++,
                });
            } else if (line.startsWith('-')) {
                parsedLines.push({
                    type: 'remove',
                    content: line.substring(1),
                    oldLineNumber: oldLineNumber++,
                });
            } else if (line.startsWith(' ')) {
                parsedLines.push({
                    type: 'context',
                    content: line.substring(1),
                    oldLineNumber: oldLineNumber++,
                    newLineNumber: newLineNumber++,
                });
            } else if (
                line.startsWith('diff ') ||
                line.startsWith('index ') ||
                line.startsWith('---') ||
                line.startsWith('+++') ||
                line.startsWith('new file') ||
                line.startsWith('deleted file')
            ) {
                parsedLines.push({
                    type: 'header',
                    content: line,
                });
            }
        }

        return parsedLines;
    };

    const identifyCollapsibleSections = (lines: DiffLine[]): CollapsibleSection[] => {
        const sections: CollapsibleSection[] = [];
        const COLLAPSE_THRESHOLD = 3; // Minimum consecutive context lines to collapse

        let currentContextStart = -1;
        let currentContextCount = 0;

        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];

            if (line.type === 'context') {
                if (currentContextStart === -1) {
                    currentContextStart = i;
                    currentContextCount = 1;
                } else {
                    currentContextCount++;
                }
            } else {
                // Non-context line encountered
                if (currentContextCount >= COLLAPSE_THRESHOLD) {
                    // Determine expand direction: check if there's a change after this section
                    const hasChangeAfter =
                        i < lines.length && (line.type === 'add' || line.type === 'remove');
                    sections.push({
                        id: `section-${currentContextStart}`,
                        startIndex: currentContextStart,
                        endIndex: currentContextStart + currentContextCount - 1,
                        totalLines: currentContextCount,
                        expandFromEnd: hasChangeAfter, // Expand from end if change is after
                    });
                }
                currentContextStart = -1;
                currentContextCount = 0;
            }
        }

        // Check if there's a trailing context section
        if (currentContextCount >= COLLAPSE_THRESHOLD) {
            sections.push({
                id: `section-${currentContextStart}`,
                startIndex: currentContextStart,
                endIndex: currentContextStart + currentContextCount - 1,
                totalLines: currentContextCount,
                expandFromEnd: false, // No changes after, expand from start
            });
        }

        return sections;
    };

    const getDiffLineClass = (type: DiffLine['type']): string => {
        switch (type) {
            case 'add':
                return 'bg-green-50 border-l-4 border-green-500 text-green-900';
            case 'remove':
                return 'bg-red-50 border-l-4 border-red-500 text-red-900';
            case 'context':
                return 'bg-gray-50';
            case 'header':
                return 'bg-blue-50 text-blue-900 font-medium';
            default:
                return '';
        }
    };

    const handleExpandSection = (sectionId: string) => {
        setRevealedLines((prev) => {
            const newMap = new Map(prev);
            const current = newMap.get(sectionId) || 0;
            newMap.set(sectionId, current + 5); // Expand 5 lines at a time
            return newMap;
        });
    };

    const diffLines = diff ? parseDiff(diff) : [];
    const collapsibleSections = identifyCollapsibleSections(diffLines);

    return (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
            <div className="bg-white rounded-lg shadow-xl max-w-6xl w-full max-h-[90vh] flex flex-col">
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b">
                    <div>
                        <h2 className="text-lg font-semibold">Changes in {fileName}</h2>
                        <p className="text-sm text-muted-foreground">{filePath}</p>
                    </div>
                    <Button variant="ghost" size="sm" onClick={onClose} className="h-8 w-8 p-0">
                        <X className="h-4 w-4" />
                    </Button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-auto">
                    {isLoading ? (
                        <div className="flex items-center justify-center h-48">
                            <div className="flex items-center gap-2 text-muted-foreground">
                                <Loader2 className="h-5 w-5 animate-spin" />
                                Loading diff...
                            </div>
                        </div>
                    ) : error ? (
                        <div className="flex items-center justify-center h-48">
                            <div className="text-center">
                                <p className="text-red-600 mb-2">Error loading diff</p>
                                <p className="text-sm text-muted-foreground">{error}</p>
                            </div>
                        </div>
                    ) : diffLines.length === 0 ? (
                        <div className="flex items-center justify-center h-48">
                            <p className="text-muted-foreground">No changes to display</p>
                        </div>
                    ) : (
                        <div className="font-mono text-sm">
                            {diffLines.map((line, index) => {
                                // Check if this line is in a collapsible section
                                const section = collapsibleSections.find(
                                    (s) => index >= s.startIndex && index <= s.endIndex
                                );

                                if (section) {
                                    const revealed = revealedLines.get(section.id) || 0;
                                    const positionInSection = index - section.startIndex;

                                    let shouldShow: boolean;
                                    let shouldShowButtonBefore = false;

                                    if (section.expandFromEnd) {
                                        // Expand from end (backwards) - show last N lines
                                        const hiddenFromStart = section.totalLines - revealed;
                                        shouldShow = positionInSection >= hiddenFromStart;
                                        // Show button before the first visible line
                                        shouldShowButtonBefore =
                                            positionInSection === hiddenFromStart &&
                                            revealed < section.totalLines;
                                    } else {
                                        // Expand from start (forward) - show first N lines
                                        shouldShow = positionInSection < revealed;
                                    }

                                    // Don't render if hidden
                                    if (!shouldShow) {
                                        // For expand-from-start sections, show button after last revealed line
                                        if (
                                            !section.expandFromEnd &&
                                            positionInSection === revealed &&
                                            revealed < section.totalLines
                                        ) {
                                            const remaining = section.totalLines - revealed;
                                            const willReveal = Math.min(5, remaining);

                                            return (
                                                <div
                                                    key={`expand-${section.id}`}
                                                    className="flex items-center justify-center py-2 bg-gray-100 border-y border-gray-200"
                                                >
                                                    <button
                                                        onClick={() =>
                                                            handleExpandSection(section.id)
                                                        }
                                                        className="flex items-center gap-2 px-4 py-1 text-sm text-gray-600 hover:text-gray-900 hover:bg-gray-200 rounded transition-colors"
                                                    >
                                                        <ChevronDown className="h-4 w-4" />
                                                        <span>
                                                            Show {willReveal} more line
                                                            {willReveal !== 1 ? 's' : ''} (
                                                            {remaining} remaining)
                                                        </span>
                                                    </button>
                                                </div>
                                            );
                                        }
                                        return null;
                                    }

                                    // Render button before this line if needed (expand from end)
                                    if (shouldShowButtonBefore) {
                                        const remaining = section.totalLines - revealed;
                                        const willReveal = Math.min(5, remaining);

                                        return (
                                            <>
                                                <div
                                                    key={`expand-${section.id}`}
                                                    className="flex items-center justify-center py-2 bg-gray-100 border-y border-gray-200"
                                                >
                                                    <button
                                                        onClick={() =>
                                                            handleExpandSection(section.id)
                                                        }
                                                        className="flex items-center gap-2 px-4 py-1 text-sm text-gray-600 hover:text-gray-900 hover:bg-gray-200 rounded transition-colors"
                                                    >
                                                        <ChevronDown className="h-4 w-4" />
                                                        <span>
                                                            Show {willReveal} more line
                                                            {willReveal !== 1 ? 's' : ''} (
                                                            {remaining} remaining)
                                                        </span>
                                                    </button>
                                                </div>
                                                <div
                                                    key={index}
                                                    className={`flex ${getDiffLineClass(
                                                        line.type
                                                    )} px-4 py-1`}
                                                >
                                                    <div className="flex-shrink-0 w-16 text-right pr-4 text-muted-foreground text-xs">
                                                        {line.type === 'add' &&
                                                            line.newLineNumber && (
                                                                <span>+{line.newLineNumber}</span>
                                                            )}
                                                        {line.type === 'remove' &&
                                                            line.oldLineNumber && (
                                                                <span>-{line.oldLineNumber}</span>
                                                            )}
                                                        {line.type === 'context' &&
                                                            line.oldLineNumber && (
                                                                <span>{line.oldLineNumber}</span>
                                                            )}
                                                    </div>
                                                    <div className="flex-1 whitespace-pre-wrap break-all">
                                                        {line.type === 'add' && (
                                                            <span className="text-green-600 mr-1">
                                                                +
                                                            </span>
                                                        )}
                                                        {line.type === 'remove' && (
                                                            <span className="text-red-600 mr-1">
                                                                -
                                                            </span>
                                                        )}
                                                        {line.type === 'context' && (
                                                            <span className="text-muted-foreground mr-1">
                                                                {' '}
                                                            </span>
                                                        )}
                                                        {line.content || ' '}
                                                    </div>
                                                </div>
                                            </>
                                        );
                                    }
                                }

                                // Render the line normally
                                return (
                                    <div
                                        key={index}
                                        className={`flex ${getDiffLineClass(line.type)} px-4 py-1`}
                                    >
                                        <div className="flex-shrink-0 w-16 text-right pr-4 text-muted-foreground text-xs">
                                            {line.type === 'add' && line.newLineNumber && (
                                                <span>+{line.newLineNumber}</span>
                                            )}
                                            {line.type === 'remove' && line.oldLineNumber && (
                                                <span>-{line.oldLineNumber}</span>
                                            )}
                                            {line.type === 'context' && line.oldLineNumber && (
                                                <span>{line.oldLineNumber}</span>
                                            )}
                                        </div>
                                        <div className="flex-1 whitespace-pre-wrap break-all">
                                            {line.type === 'add' && (
                                                <span className="text-green-600 mr-1">+</span>
                                            )}
                                            {line.type === 'remove' && (
                                                <span className="text-red-600 mr-1">-</span>
                                            )}
                                            {line.type === 'context' && (
                                                <span className="text-muted-foreground mr-1">
                                                    {' '}
                                                </span>
                                            )}
                                            {line.content || ' '}
                                        </div>
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
