import { useState, useEffect } from 'react';
import { useAtom } from 'jotai';
import { selectedRepositoryAtom, vimModeEnabledAtom, vimFocusContextAtom, vimFocusIndexAtom } from '@/store/atoms';
import { useCommitHistory, useRepositoryStatus, usePush } from '@/store/queries';
import { format } from 'date-fns';
import { GitCommit, User, Calendar, Hash, Upload, AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
    DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu';
import CommitDetailsDialog from './CommitDetailsDialog';
import { useVimNavigation } from '@/hooks/use-vim-navigation';

export default function CommitHistory() {
    const [currentRepository] = useAtom(selectedRepositoryAtom);
    const { data: commits, isLoading, error } = useCommitHistory(currentRepository?.id);
    const { data: status } = useRepositoryStatus(currentRepository?.id);
    const [selectedCommitHash, setSelectedCommitHash] = useState<string | null>(null);
    const [showForcePushWarning, setShowForcePushWarning] = useState(false);
    const pushMutation = usePush();

    // Vim navigation
    const [vimEnabled] = useAtom(vimModeEnabledAtom);
    const [vimContext, setVimContext] = useAtom(vimFocusContextAtom);
    const [vimIndex] = useAtom(vimFocusIndexAtom);

    const { isVimActive, currentIndex, activateContext } = useVimNavigation({
        context: 'commit-list',
        itemCount: commits?.length || 0,
        onActivate: (index) => {
            if (commits && commits[index]) {
                setSelectedCommitHash(commits[index].hash);
            }
        },
    });

    // Auto-activate this context when vim is enabled and we're in this view
    useEffect(() => {
        if (vimEnabled && commits && commits.length > 0 && vimContext === 'none') {
            activateContext();
        }
    }, [vimEnabled, commits, vimContext, activateContext]);

    // Handle clicking on the container to activate vim context
    const handleContainerClick = () => {
        if (vimEnabled && commits && commits.length > 0) {
            setVimContext('commit-list');
        }
    };

    // Push handlers
    const handlePush = () => {
        if (currentRepository) {
            pushMutation.mutate({ repositoryId: currentRepository.id, force: false });
        }
    };

    const handleForcePush = () => {
        if (currentRepository) {
            pushMutation.mutate({ repositoryId: currentRepository.id, force: true });
            setShowForcePushWarning(false);
        }
    };

    // Debug logging
    console.log('CommitHistory - currentRepository:', currentRepository);
    console.log('CommitHistory - commits:', commits);
    console.log('CommitHistory - isLoading:', isLoading);
    console.log('CommitHistory - error:', error);

    if (isLoading) {
        return (
            <div className="p-4 flex items-center justify-center">
                <div className="text-muted-foreground">Loading commit history...</div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-4 flex items-center justify-center">
                <div className="text-red-600">Error loading commits: {error.message}</div>
            </div>
        );
    }

    if (!commits || commits.length === 0) {
        return (
            <div className="p-4 flex items-center justify-center">
                <div className="text-center text-muted-foreground">
                    <GitCommit className="h-12 w-12 mx-auto mb-2 opacity-50" />
                    <h3 className="text-lg font-medium mb-1">No Commits Yet</h3>
                    <p>This repository doesn't have any commits yet</p>
                </div>
            </div>
        );
    }

    return (
        <div className="h-full flex flex-col">
            <div className="p-4 border-b bg-muted/50">
                <div className="flex items-center justify-between">
                    <h2 className="text-lg font-semibold flex items-center gap-2">
                        <GitCommit className="h-5 w-5" />
                        Commit History
                    </h2>
                    <div className="flex items-center gap-3">
                        <div className="text-sm text-muted-foreground">
                            {commits.length} commits
                        </div>
                        {status && status.ahead > 0 && (
                            <div className="flex items-center gap-2">
                                <Badge variant="secondary" className="flex items-center gap-1">
                                    <Upload className="h-3 w-3" />
                                    {status.ahead} to push
                                </Badge>
                                <DropdownMenu>
                                    <DropdownMenuTrigger asChild>
                                        <Button
                                            size="sm"
                                            disabled={pushMutation.isPending}
                                            className="h-8"
                                        >
                                            <Upload className="h-4 w-4 mr-2" />
                                            {pushMutation.isPending ? 'Pushing...' : 'Push'}
                                        </Button>
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="end">
                                        <DropdownMenuItem onClick={handlePush}>
                                            Push to remote
                                        </DropdownMenuItem>
                                        <DropdownMenuSeparator />
                                        <DropdownMenuItem
                                            onClick={() => setShowForcePushWarning(true)}
                                            className="text-red-600 focus:text-red-600"
                                        >
                                            <AlertTriangle className="h-4 w-4 mr-2" />
                                            Force push
                                        </DropdownMenuItem>
                                    </DropdownMenuContent>
                                </DropdownMenu>
                            </div>
                        )}
                    </div>
                </div>
            </div>

            <div className="flex-1 overflow-auto" onClick={handleContainerClick}>
                <div className="space-y-2 p-4">
                    {commits.map((commit, index) => (
                        <div
                            key={commit.hash}
                            className={`border rounded-lg p-4 hover:bg-muted/50 transition-colors cursor-pointer ${
                                isVimActive && currentIndex === index ? 'ring-2 ring-blue-500 bg-blue-50/50' : ''
                            }`}
                        >
                            <div className="flex items-start gap-3">
                                <div className="flex-shrink-0 mt-1">
                                    <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
                                        <GitCommit className="h-4 w-4 text-primary" />
                                    </div>
                                </div>
                                
                                <div className="flex-1 min-w-0">
                                    <div className="flex items-start justify-between gap-4">
                                        <div className="flex-1 min-w-0">
                                            <h3 className="font-medium text-sm leading-5 mb-2 break-words">
                                                {commit.message}
                                            </h3>
                                            
                                            <div className="flex items-center gap-4 text-xs text-muted-foreground">
                                                <div className="flex items-center gap-1">
                                                    <User className="h-3 w-3" />
                                                    <span>{commit.author.name}</span>
                                                </div>
                                                <div className="flex items-center gap-1">
                                                    <Calendar className="h-3 w-3" />
                                                    <span>{format(new Date(commit.date), 'MMM d, yyyy HH:mm')}</span>
                                                </div>
                                            </div>
                                        </div>
                                        
                                        <div className="flex items-center gap-2 flex-shrink-0">
                                            <div className="flex items-center gap-1 text-xs text-muted-foreground bg-muted px-2 py-1 rounded font-mono">
                                                <Hash className="h-3 w-3" />
                                                <span>{commit.hash.substring(0, 7)}</span>
                                            </div>
                                            <Button 
                                                variant="ghost" 
                                                size="sm" 
                                                className="h-6 px-2"
                                                onClick={() => setSelectedCommitHash(commit.hash)}
                                            >
                                                View
                                            </Button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            </div>

            {/* Force Push Warning */}
            {showForcePushWarning && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
                    <div className="bg-background border rounded-lg shadow-lg max-w-md w-full p-6">
                        <Alert variant="destructive" className="mb-4">
                            <AlertTriangle className="h-4 w-4" />
                            <AlertTitle>Warning: Force Push</AlertTitle>
                            <AlertDescription>
                                Force pushing will overwrite the remote history. This action cannot be undone
                                and may cause problems for other collaborators.
                            </AlertDescription>
                        </Alert>
                        <div className="flex justify-end gap-2">
                            <Button
                                variant="outline"
                                onClick={() => setShowForcePushWarning(false)}
                            >
                                Cancel
                            </Button>
                            <Button
                                variant="destructive"
                                onClick={handleForcePush}
                                disabled={pushMutation.isPending}
                            >
                                {pushMutation.isPending ? 'Force Pushing...' : 'Force Push'}
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            <CommitDetailsDialog
                commitHash={selectedCommitHash}
                isOpen={!!selectedCommitHash}
                onClose={() => setSelectedCommitHash(null)}
            />
        </div>
    );
}