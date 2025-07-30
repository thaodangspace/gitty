import { useState } from 'react';
import { useAtom } from 'jotai';
import { selectedRepositoryAtom, selectedFilesAtom } from '@/store/atoms';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { useIsMobile } from '@/hooks/use-mobile';
import { Button } from '@/components/ui/button';
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
}

export default function FileTreeBrowser({ onFileSelect }: FileTreeBrowserProps = {}) {
    const [currentRepository] = useAtom(selectedRepositoryAtom);
    const [selectedFiles, setSelectedFiles] = useAtom(selectedFilesAtom);
    const [expandedDirs, setExpandedDirs] = useState<Set<string>>(new Set());
    const isMobile = useIsMobile();

    const { data: files, isLoading, error } = useQuery({
        queryKey: ['file-tree', currentRepository?.id],
        queryFn: () => apiClient.getFileTree(currentRepository!.id),
        enabled: !!currentRepository?.id,
    });

    const buildFileTree = (files: FileInfo[]): TreeNode[] => {
        const root: TreeNode[] = [];
        const map = new Map<string, TreeNode>();

        // Sort files - directories first, then by name
        const sortedFiles = [...(files || [])].sort((a, b) => {
            if (a.is_directory !== b.is_directory) {
                return a.is_directory ? -1 : 1;
            }
            return a.name.localeCompare(b.name);
        });

        // Create tree structure
        sortedFiles.forEach(file => {
            const pathParts = file.path.split('/').filter(Boolean);
            let currentLevel = root;
            let currentPath = '';
            
            pathParts.forEach((part, index) => {
                currentPath += (currentPath ? '/' : '') + part;
                const isLast = index === pathParts.length - 1;
                
                if (isLast) {
                    // This is the actual file/directory from the backend
                    // Check if a node with this path already exists (from intermediate directory creation)
                    let existingNode = map.get(file.path);
                    if (existingNode) {
                        // Update the existing node with real data from backend
                        existingNode.size = file.size;
                        existingNode.mod_time = file.mod_time;
                        existingNode.mode = file.mode;
                        existingNode.level = index;
                    } else {
                        // Create new node
                        const node: TreeNode = {
                            ...file,
                            level: index,
                            children: file.is_directory ? [] : undefined,
                            isExpanded: expandedDirs.has(file.path)
                        };
                        currentLevel.push(node);
                        map.set(file.path, node);
                    }
                } else {
                    // This is a parent directory we need to create if it doesn't exist
                    let existingNode = currentLevel.find(n => n.name === part);
                    if (!existingNode) {
                        existingNode = {
                            path: currentPath,
                            name: part,
                            is_directory: true,
                            size: 0,
                            mod_time: '',
                            mode: '',
                            level: index,
                            children: [],
                            isExpanded: expandedDirs.has(currentPath)
                        };
                        currentLevel.push(existingNode);
                        map.set(currentPath, existingNode);
                    }
                    currentLevel = existingNode.children!;
                }
            });
        });

        return root;
    };

    const toggleDirectory = (path: string) => {
        setExpandedDirs(prev => {
            const newSet = new Set(prev);
            if (newSet.has(path)) {
                newSet.delete(path);
            } else {
                newSet.add(path);
            }
            return newSet;
        });
    };

    const selectFile = (file: TreeNode) => {
        if (file.is_directory) {
            toggleDirectory(file.path);
        } else {
            setSelectedFiles([file.path]);
            // Close mobile drawer when file is selected
            if (isMobile && onFileSelect) {
                onFileSelect();
            }
        }
    };

    const renderTreeNode = (node: TreeNode) => {
        const isSelected = selectedFiles.includes(node.path);
        const paddingLeft = node.level * 16 + 8;

        return (
            <div key={node.path}>
                <div
                    className={`flex items-center py-1 px-2 hover:bg-muted/50 cursor-pointer text-sm
                        ${isSelected ? 'bg-primary/10 text-primary' : ''}
                        ${isMobile ? 'touch-target py-3' : ''}
                    `}
                    style={{ paddingLeft }}
                    onClick={() => selectFile(node)}
                >
                    {node.is_directory && (
                        <Button variant="ghost" size="sm" className="h-4 w-4 p-0 mr-1">
                            {node.isExpanded ? (
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
                
                {node.is_directory && node.isExpanded && node.children && (
                    <div>
                        {node.children.map(child => renderTreeNode(child))}
                    </div>
                )}
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

    if (isLoading) {
        return (
            <div className="h-full flex items-center justify-center">
                <div className="flex items-center gap-2 text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    <span className="text-sm">Loading files...</span>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="h-full flex items-center justify-center">
                <div className="flex items-center gap-2 text-red-500">
                    <AlertCircle className="h-4 w-4" />
                    <span className="text-sm">Failed to load files</span>
                </div>
            </div>
        );
    }

    const fileTree = buildFileTree(files || []);

    return (
        <div className="h-full overflow-auto">
            <div className="p-2 border-b">
                <h3 className="font-medium text-sm">Files</h3>
            </div>
            <div className="py-2">
                {fileTree.length === 0 ? (
                    <div className="p-4 text-center text-muted-foreground text-sm">
                        No files found
                    </div>
                ) : (
                    fileTree.map(node => renderTreeNode(node))
                )}
            </div>
        </div>
    );
}