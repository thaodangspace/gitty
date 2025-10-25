import { useState, useEffect, useMemo } from 'react';
import { useAtom } from 'jotai';
import { selectedRepositoryAtom, vimModeEnabledAtom, vimFocusContextAtom, vimFocusIndexAtom } from '@/store/atoms';
import { useRepositoryStatus, useStageFile, useUnstageFile } from '@/store/queries';
import type { FileChange } from '@/types/api';
import { useVimNavigation } from '@/hooks/use-vim-navigation';
import { 
    GitBranch, 
    FileText, 
    FilePlus, 
    FileX, 
    FileMinus, 
    Plus,
    Minus,
    Clock,
    CheckCircle2,
    Circle,
    GitCommit
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import CommitDialog from './CommitDialog';
import DiffViewer from '../file/DiffViewer';

const getStatusIcon = (status: string) => {
    switch (status) {
        case 'A':
            return <FilePlus className="h-4 w-4 text-green-600" />;
        case 'M':
            return <FileText className="h-4 w-4 text-orange-600" />;
        case 'D':
            return <FileX className="h-4 w-4 text-red-600" />;
        case 'R':
            return <FileMinus className="h-4 w-4 text-blue-600" />;
        default:
            return <FileText className="h-4 w-4 text-muted-foreground" />;
    }
};

const getStatusText = (status: string) => {
    switch (status) {
        case 'A':
            return 'Added';
        case 'M':
            return 'Modified';
        case 'D':
            return 'Deleted';
        case 'R':
            return 'Renamed';
        case '?':
            return 'Untracked';
        default:
            return status;
    }
};

const getStatusColor = (status: string) => {
    switch (status) {
        case 'A':
            return 'text-green-600 bg-green-50 border-green-200';
        case 'M':
            return 'text-orange-600 bg-orange-50 border-orange-200';
        case 'D':
            return 'text-red-600 bg-red-50 border-red-200';
        case 'R':
            return 'text-blue-600 bg-blue-50 border-blue-200';
        default:
            return 'text-muted-foreground bg-muted border-border';
    }
};

interface FileChangeItemProps {
    file: FileChange;
    isStaged?: boolean;
    onStage?: () => void;
    onUnstage?: () => void;
    onViewDiff?: () => void;
    isVimFocused?: boolean;
}

function FileChangeItem({ file, isStaged = false, onStage, onUnstage, onViewDiff, isVimFocused = false }: FileChangeItemProps) {
    return (
        <div className={`flex items-center gap-3 p-3 border rounded-lg hover:bg-muted/50 transition-colors ${
            isVimFocused ? 'ring-2 ring-blue-500 bg-blue-50/50' : ''
        }`}>
            <div className="flex-shrink-0">
                {getStatusIcon(file.status)}
            </div>
            
            <div 
                className="flex-1 min-w-0 cursor-pointer" 
                onClick={onViewDiff}
                title="Click to view changes"
            >
                <div className="flex items-center gap-2">
                    <span className="font-medium text-sm truncate hover:text-primary">{file.path}</span>
                    <Badge variant="outline" className={`h-5 text-xs ${getStatusColor(file.status)}`}>
                        {getStatusText(file.status)}
                    </Badge>
                </div>
            </div>
            
            <div className="flex-shrink-0 flex items-center gap-1">
                <Button
                    variant="ghost"
                    size="sm"
                    onClick={onViewDiff}
                    className="h-7 px-2 text-muted-foreground hover:text-foreground"
                    title="View changes"
                >
                    <FileText className="h-3 w-3" />
                </Button>
                {isStaged ? (
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={onUnstage}
                        className="h-7 px-2 text-muted-foreground hover:text-foreground"
                    >
                        <Minus className="h-3 w-3 mr-1" />
                        Unstage
                    </Button>
                ) : (
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={onStage}
                        className="h-7 px-2 text-muted-foreground hover:text-foreground"
                    >
                        <Plus className="h-3 w-3 mr-1" />
                        Stage
                    </Button>
                )}
            </div>
        </div>
    );
}

interface UntrackedFileItemProps {
    fileName: string;
    onStage?: () => void;
    isVimFocused?: boolean;
}

function UntrackedFileItem({ fileName, onStage, isVimFocused = false }: UntrackedFileItemProps) {
    return (
        <div className={`flex items-center gap-3 p-3 border rounded-lg hover:bg-muted/50 transition-colors ${
            isVimFocused ? 'ring-2 ring-blue-500 bg-blue-50/50' : ''
        }`}>
            <div className="flex-shrink-0">
                <Circle className="h-4 w-4 text-muted-foreground" />
            </div>
            
            <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                    <span className="font-medium text-sm text-muted-foreground truncate">{fileName}</span>
                    <Badge variant="outline" className="h-5 text-xs text-muted-foreground">
                        Untracked
                    </Badge>
                </div>
            </div>
            
            <div className="flex-shrink-0">
                <Button
                    variant="ghost"
                    size="sm"
                    onClick={onStage}
                    className="h-7 px-2 text-muted-foreground hover:text-foreground"
                >
                    <Plus className="h-3 w-3 mr-1" />
                    Stage
                </Button>
            </div>
        </div>
    );
}

export default function WorkingDirectoryChanges() {
    const [currentRepository] = useAtom(selectedRepositoryAtom);
    const { data: repoStatus, isLoading, error } = useRepositoryStatus(currentRepository?.id);
    const stageFileMutation = useStageFile();
    const unstageFileMutation = useUnstageFile();
    const [showCommitDialog, setShowCommitDialog] = useState(false);
    const [selectedDiffFile, setSelectedDiffFile] = useState<{ path: string; name: string } | null>(null);

    // Vim navigation
    const [vimEnabled] = useAtom(vimModeEnabledAtom);
    const [vimContext, setVimContext] = useAtom(vimFocusContextAtom);
    const [vimIndex] = useAtom(vimFocusIndexAtom);

    // Create flattened list of all files for vim navigation
    const allFiles = useMemo(() => {
        if (!repoStatus) return [];

        const files: Array<{ type: 'staged' | 'modified' | 'untracked'; data: FileChange | string }> = [];

        repoStatus.staged.forEach(file => files.push({ type: 'staged', data: file }));
        repoStatus.modified.forEach(file => files.push({ type: 'modified', data: file }));
        repoStatus.untracked.forEach(file => files.push({ type: 'untracked', data: file }));

        return files;
    }, [repoStatus]);

    const { isVimActive, currentIndex, activateContext } = useVimNavigation({
        context: 'file-changes',
        itemCount: allFiles.length,
        onActivate: (index) => {
            const file = allFiles[index];
            if (file.type === 'staged' || file.type === 'modified') {
                const fileChange = file.data as FileChange;
                handleViewDiff(fileChange.path);
            }
        },
    });

    // Auto-activate this context when vim is enabled and we're in this view
    useEffect(() => {
        if (vimEnabled && allFiles.length > 0 && vimContext === 'none') {
            activateContext();
        }
    }, [vimEnabled, allFiles.length, vimContext, activateContext]);

    // Handle clicking on the container to activate vim context
    const handleContainerClick = () => {
        if (vimEnabled && allFiles.length > 0) {
            setVimContext('file-changes');
        }
    };

    // Debug logging
    console.log('WorkingDirectoryChanges - currentRepository:', currentRepository);
    console.log('WorkingDirectoryChanges - repoStatus:', repoStatus);
    console.log('WorkingDirectoryChanges - isLoading:', isLoading);
    console.log('WorkingDirectoryChanges - error:', error);

    const handleStageFile = (filePath: string) => {
        if (!currentRepository) return;
        
        stageFileMutation.mutate({
            repositoryId: currentRepository.id,
            filePath,
        });
    };

    const handleUnstageFile = (filePath: string) => {
        if (!currentRepository) return;
        
        unstageFileMutation.mutate({
            repositoryId: currentRepository.id,
            filePath,
        });
    };

    const handleViewDiff = (filePath: string) => {
        const fileName = filePath.split('/').pop() || filePath;
        setSelectedDiffFile({ path: filePath, name: fileName });
    };

    if (isLoading) {
        return (
            <div className="p-4 flex items-center justify-center">
                <div className="text-muted-foreground">Loading repository status...</div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-4 flex items-center justify-center">
                <div className="text-red-600">Error loading status: {error.message}</div>
            </div>
        );
    }

    if (!repoStatus) {
        return (
            <div className="p-4 flex items-center justify-center">
                <div className="text-muted-foreground">No repository status available</div>
            </div>
        );
    }

    const totalChanges = repoStatus.staged.length + repoStatus.modified.length + repoStatus.untracked.length;

    if (totalChanges === 0) {
        return (
            <div className="h-full flex items-center justify-center">
                <div className="text-center text-muted-foreground">
                    <CheckCircle2 className="h-12 w-12 mx-auto mb-2 opacity-50" />
                    <h3 className="text-lg font-medium mb-1">Working Directory Clean</h3>
                    <p>No changes to display</p>
                </div>
            </div>
        );
    }

    return (
        <div className="h-full flex flex-col">
            <div className="p-4 border-b bg-muted/50">
                <div className="flex items-center justify-between">
                    <h2 className="text-lg font-semibold flex items-center gap-2">
                        <GitBranch className="h-5 w-5" />
                        Changes
                    </h2>
                    <div className="flex items-center gap-2">
                        <Badge variant="secondary" className="h-6">
                            {repoStatus.branch}
                        </Badge>
                        <div className="text-sm text-muted-foreground">
                            {totalChanges} {totalChanges === 1 ? 'change' : 'changes'}
                        </div>
                        {repoStatus.staged.length > 0 && (
                            <Button 
                                size="sm" 
                                className="h-7"
                                onClick={() => setShowCommitDialog(true)}
                            >
                                <GitCommit className="h-3 w-3 mr-1" />
                                Commit ({repoStatus.staged.length})
                            </Button>
                        )}
                    </div>
                </div>
            </div>

            <div className="flex-1 overflow-auto" onClick={handleContainerClick}>
                <div className="p-4 space-y-6">
                    {/* Staged Changes */}
                    {repoStatus.staged.length > 0 && (
                        <div>
                            <div className="flex items-center gap-2 mb-3">
                                <CheckCircle2 className="h-4 w-4 text-green-600" />
                                <h3 className="text-sm font-semibold text-green-600">
                                    Staged Changes ({repoStatus.staged.length})
                                </h3>
                            </div>
                            <div className="space-y-2">
                                {repoStatus.staged.map((file, index) => {
                                    const globalIndex = index;
                                    return (
                                        <FileChangeItem
                                            key={`staged-${index}`}
                                            file={file}
                                            isStaged={true}
                                            onUnstage={() => handleUnstageFile(file.path)}
                                            onViewDiff={() => handleViewDiff(file.path)}
                                            isVimFocused={isVimActive && currentIndex === globalIndex}
                                        />
                                    );
                                })}
                            </div>
                        </div>
                    )}

                    {/* Modified Files */}
                    {repoStatus.modified.length > 0 && (
                        <div>
                            <div className="flex items-center gap-2 mb-3">
                                <Clock className="h-4 w-4 text-orange-600" />
                                <h3 className="text-sm font-semibold text-orange-600">
                                    Modified Files ({repoStatus.modified.length})
                                </h3>
                            </div>
                            <div className="space-y-2">
                                {repoStatus.modified.map((file, index) => {
                                    const globalIndex = repoStatus.staged.length + index;
                                    return (
                                        <FileChangeItem
                                            key={`modified-${index}`}
                                            file={file}
                                            isStaged={false}
                                            onStage={() => handleStageFile(file.path)}
                                            onViewDiff={() => handleViewDiff(file.path)}
                                            isVimFocused={isVimActive && currentIndex === globalIndex}
                                        />
                                    );
                                })}
                            </div>
                        </div>
                    )}

                    {/* Untracked Files */}
                    {repoStatus.untracked.length > 0 && (
                        <div>
                            <div className="flex items-center gap-2 mb-3">
                                <Circle className="h-4 w-4 text-muted-foreground" />
                                <h3 className="text-sm font-semibold text-muted-foreground">
                                    Untracked Files ({repoStatus.untracked.length})
                                </h3>
                            </div>
                            <div className="space-y-2">
                                {repoStatus.untracked.map((fileName, index) => {
                                    const globalIndex = repoStatus.staged.length + repoStatus.modified.length + index;
                                    return (
                                        <UntrackedFileItem
                                            key={`untracked-${index}`}
                                            fileName={fileName}
                                            onStage={() => handleStageFile(fileName)}
                                            isVimFocused={isVimActive && currentIndex === globalIndex}
                                        />
                                    );
                                })}
                            </div>
                        </div>
                    )}

                    {/* Conflicts */}
                    {repoStatus.conflicts.length > 0 && (
                        <div>
                            <div className="flex items-center gap-2 mb-3">
                                <FileX className="h-4 w-4 text-red-600" />
                                <h3 className="text-sm font-semibold text-red-600">
                                    Conflicts ({repoStatus.conflicts.length})
                                </h3>
                            </div>
                            <div className="space-y-2">
                                {repoStatus.conflicts.map((fileName, index) => (
                                    <div
                                        key={`conflict-${index}`}
                                        className="flex items-center gap-3 p-3 border border-red-200 bg-red-50 rounded-lg"
                                    >
                                        <FileX className="h-4 w-4 text-red-600" />
                                        <span className="font-medium text-sm text-red-800">{fileName}</span>
                                        <Button variant="outline" size="sm" className="ml-auto">
                                            Resolve
                                        </Button>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            </div>
            
            <CommitDialog 
                open={showCommitDialog} 
                onOpenChange={setShowCommitDialog} 
            />

            {/* Diff Viewer */}
            {selectedDiffFile && currentRepository && (
                <DiffViewer
                    repositoryId={currentRepository.id}
                    filePath={selectedDiffFile.path}
                    fileName={selectedDiffFile.name}
                    onClose={() => setSelectedDiffFile(null)}
                />
            )}
        </div>
    );
}