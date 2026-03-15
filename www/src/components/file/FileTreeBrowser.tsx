import { useState, useMemo, useCallback } from 'react';
import { useAtom } from 'jotai';
import { selectedRepositoryAtom, selectedFilesAtom, vimModeEnabledAtom, vimFocusContextAtom } from '@/store/atoms';
import { useQueries } from '@tanstack/react-query';
import { useRepoDirectoryListing } from '@/hooks/api';
import { apiClient } from '@/lib/api-client';
import { useIsMobile } from '@/hooks/use-mobile';
import { Button } from '@/components/ui/button';
import { useVimNavigation } from '@/hooks/use-vim-navigation';
import {
    ChevronRight,
    ChevronDown,
    File,
    Folder,
    FolderOpen,
    Loader2,
    AlertCircle
} from 'lucide-react';
import type { FileInfo } from '@/types/api';

interface FileTreeBrowserProps {
    onFileSelect?: () => void;
}

interface TreeNode extends FileInfo {
    children?: TreeNode[];
    isExpanded?: boolean;
    level: number;
    isLoading?: boolean;
}

export default function FileTreeBrowser({ onFileSelect }: FileTreeBrowserProps = {}) {
    const [currentRepository] = useAtom(selectedRepositoryAtom);
    const [selectedFiles, setSelectedFiles] = useAtom(selectedFilesAtom);
    const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());
    const [filterText, setFilterText] = useState('');
    const isMobile = useIsMobile();

    // Vim navigation
    const [vimEnabled] = useAtom(vimModeEnabledAtom);
    const [, setVimContext] = useAtom(vimFocusContextAtom);

    // Load root directory
    const rootQuery = useRepoDirectoryListing(currentRepository?.id || '', '');

    // Load each expanded directory using useQueries
    const expandedQueries = useQueries({
        queries: Array.from(expandedPaths).map(path => ({
            queryKey: ['repository', currentRepository?.id, 'directory', path],
            queryFn: () => apiClient.browseRepoDirectory(currentRepository!.id, path),
            enabled: !!currentRepository?.id && expandedPaths.has(path),
            staleTime: 30_000,
        })),
    });

    // Build a map of path -> entries from queries
    const directoryMap = useMemo(() => {
        const map = new Map<string, FileInfo[]>();

        if (rootQuery.data?.entries) {
            map.set('', rootQuery.data.entries);
        }

        const expandedPathsArray = Array.from(expandedPaths);
        expandedPathsArray.forEach((path, index) => {
            const query = expandedQueries[index];
            if (query.data?.entries) {
                map.set(path, query.data.entries);
            }
        });

        return map;
    }, [rootQuery.data, expandedQueries, expandedPaths]);

    // Build file tree from directory map
    const buildFileTree = useCallback((entries: FileInfo[] | undefined, level: number): TreeNode[] => {
        if (!Array.isArray(entries)) return [];
        return entries.map(entry => {
            const isExpanded = expandedPaths.has(entry.path);
            const children = isExpanded && directoryMap.has(entry.path)
                ? buildFileTree(directoryMap.get(entry.path)!, level + 1)
                : undefined;

            return {
                ...entry,
                children,
                isExpanded,
                level,
                isLoading: isExpanded && !directoryMap.has(entry.path),
            };
        });
    }, [directoryMap, expandedPaths]);

    const rootEntries = rootQuery.data?.entries || [];
    const fileTree = useMemo(() => buildFileTree(rootEntries, 0), [rootEntries, buildFileTree]);

    // Flatten tree for vim navigation
    const flattenTree = useCallback((nodes: TreeNode[]): TreeNode[] => {
        const flattened: TreeNode[] = [];
        const traverse = (nodeList: TreeNode[]) => {
            nodeList.forEach(node => {
                flattened.push(node);
                if (node.is_directory && node.isExpanded && node.children) {
                    traverse(node.children);
                }
            });
        };
        traverse(nodes);
        return flattened;
    }, []);

    const flattenedNodes = useMemo(() => flattenTree(fileTree), [fileTree, flattenTree]);

    // Toggle directory expansion
    const toggleDirectory = useCallback((path: string) => {
        setExpandedPaths(prev => {
            const newSet = new Set(prev);
            if (newSet.has(path)) {
                newSet.delete(path);
            } else {
                newSet.add(path);
            }
            return newSet;
        });
    }, []);

    // Select file
    const selectFile = useCallback((node: TreeNode) => {
        if (node.is_directory) {
            toggleDirectory(node.path);
        } else {
            setSelectedFiles([node.path]);
            if (isMobile && onFileSelect) {
                onFileSelect();
            }
        }
    }, [toggleDirectory, setSelectedFiles, isMobile, onFileSelect]);

    // Vim navigation setup
    const { isVimActive, currentIndex } = useVimNavigation({
        context: 'file-tree',
        itemCount: flattenedNodes.length,
        onActivate: (index) => {
            if (flattenedNodes[index]) {
                selectFile(flattenedNodes[index]);
            }
        },
        onExpand: (index) => {
            const node = flattenedNodes[index];
            if (node && node.is_directory && !node.isExpanded) {
                toggleDirectory(node.path);
            }
        },
        onCollapse: (index) => {
            const node = flattenedNodes[index];
            if (node && node.is_directory && node.isExpanded) {
                toggleDirectory(node.path);
            }
        },
    });

    // Handle container click for vim activation
    const handleContainerClick = useCallback(() => {
        if (vimEnabled && flattenedNodes.length > 0) {
            setVimContext('file-tree');
        }
    }, [vimEnabled, flattenedNodes.length, setVimContext]);

    // Filter entries
    const filterEntries = useCallback((entries: TreeNode[]): TreeNode[] => {
        if (!filterText) return entries;
        return entries.filter(e =>
            e.name.toLowerCase().includes(filterText.toLowerCase())
        );
    }, [filterText]);

    // Render tree node
    const renderTreeNode = (node: TreeNode, index: number) => {
        const isSelected = selectedFiles.includes(node.path);
        const isVimFocused = isVimActive && currentIndex === index;
        const paddingLeft = node.level * 16 + 8;

        return (
            <div key={node.path}>
                <div
                    className={`flex items-center py-1 px-2 hover:bg-muted/50 cursor-pointer text-sm
                        ${isSelected ? 'bg-primary/10 text-primary' : ''}
                        ${isVimFocused ? 'ring-2 ring-blue-500 bg-blue-50/50' : ''}
                        ${isMobile ? 'touch-target py-3' : ''}
                    `}
                    style={{ paddingLeft }}
                    onClick={() => selectFile(node)}
                >
                    {node.is_directory && (
                        <Button variant="ghost" size="sm" className="h-4 w-4 p-0 mr-1">
                            {node.isLoading ? (
                                <Loader2 className="h-3 w-3 animate-spin" />
                            ) : node.isExpanded ? (
                                <ChevronDown className="h-3 w-3" />
                            ) : (
                                <ChevronRight className="h-3 w-3" />
                            )}
                        </Button>
                    )}

                    {!node.is_directory && <div className="w-5" />}

                    <div className="flex items-center gap-1 flex-1 min-w-0">
                        {node.is_directory ? (
                            node.isExpanded ? (
                                <FolderOpen className="h-4 w-4 flex-shrink-0" />
                            ) : (
                                <Folder className="h-4 w-4 flex-shrink-0" />
                            )
                        ) : (
                            <File className="h-4 w-4 flex-shrink-0" />
                        )}
                        <span className="truncate">{node.name}</span>
                    </div>
                </div>
            </div>
        );
    };

    if (!currentRepository) {
        return (
            <div className="h-full flex items-center justify-center text-muted-foreground">
                <p className="text-sm">No repository selected</p>
            </div>
        );
    }

    if (rootQuery.isLoading) {
        return (
            <div className="h-full flex items-center justify-center">
                <div className="flex items-center gap-2 text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    <span className="text-sm">Loading files...</span>
                </div>
            </div>
        );
    }

    if (rootQuery.error) {
        return (
            <div className="h-full flex items-center justify-center">
                <div className="flex items-center gap-2 text-red-500">
                    <AlertCircle className="h-4 w-4" />
                    <span className="text-sm">Failed to load files</span>
                </div>
            </div>
        );
    }

    const filteredNodes = filterText ? filterEntries(flattenedNodes) : flattenedNodes;

    return (
        <div className="h-full overflow-auto" onClick={handleContainerClick}>
            <div className="p-2 border-b">
                <div className="flex items-center justify-between mb-2">
                    <h3 className="font-medium text-sm">Files</h3>
                </div>
                <input
                    type="text"
                    placeholder="Filter files..."
                    value={filterText}
                    onChange={(e) => setFilterText(e.target.value)}
                    className="w-full px-2 py-1 text-sm border rounded"
                />
            </div>
            <div className="py-2">
                {filteredNodes.length === 0 ? (
                    <div className="p-4 text-center text-muted-foreground text-sm">
                        {filterText ? 'No matching files' : 'No files found'}
                    </div>
                ) : (
                    filteredNodes.map((node, index) => renderTreeNode(node, index))
                )}
            </div>
        </div>
    );
}