import { useState } from 'react';
import { useAtom } from 'jotai';
import { selectedRepositoryFromListAtom } from '@/store/atoms';
import { useBranches, useSwitchBranch, useDeleteBranch } from '@/store/queries';
import { format } from 'date-fns';
import { GitBranch, GitMerge, Plus, Check, Hash, User, Calendar, Loader2, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { 
    Dialog, 
    DialogContent, 
    DialogDescription, 
    DialogFooter, 
    DialogHeader, 
    DialogTitle 
} from '@/components/ui/dialog';
import CreateBranchDialog from './CreateBranchDialog';

export default function BranchList() {
    const [currentRepository] = useAtom(selectedRepositoryFromListAtom);
    const { data: branches, isLoading, error } = useBranches(currentRepository?.id);
    const [showCreateDialog, setShowCreateDialog] = useState(false);
    const [deleteBranchName, setDeleteBranchName] = useState<string | null>(null);
    const switchBranchMutation = useSwitchBranch();
    const deleteBranchMutation = useDeleteBranch();

    if (isLoading) {
        return (
            <div className="p-4 flex items-center justify-center">
                <div className="text-muted-foreground">Loading branches...</div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-4 flex items-center justify-center">
                <div className="text-red-600">Error loading branches: {error.message}</div>
            </div>
        );
    }

    if (!branches || branches.length === 0) {
        return (
            <div className="p-4 flex items-center justify-center">
                <div className="text-center text-muted-foreground">
                    <GitBranch className="h-12 w-12 mx-auto mb-2 opacity-50" />
                    <h3 className="text-lg font-medium mb-1">No Branches Found</h3>
                    <p>This repository doesn't have any branches</p>
                </div>
            </div>
        );
    }

    const currentBranch = branches.find(branch => branch.is_current);
    const otherBranches = branches.filter(branch => !branch.is_current);

    const handleSwitchBranch = (branchName: string) => {
        if (!currentRepository) return;
        
        switchBranchMutation.mutate({
            repositoryId: currentRepository.id,
            branchName,
        });
    };

    const handleDeleteBranch = () => {
        if (!currentRepository || !deleteBranchName) return;
        
        deleteBranchMutation.mutate({
            repositoryId: currentRepository.id,
            branchName: deleteBranchName,
        }, {
            onSuccess: () => {
                setDeleteBranchName(null);
            }
        });
    };

    return (
        <div className="h-full flex flex-col">
            <div className="p-4 border-b bg-muted/50">
                <div className="flex items-center justify-between">
                    <h2 className="text-lg font-semibold flex items-center gap-2">
                        <GitMerge className="h-5 w-5" />
                        Branches
                    </h2>
                    <div className="flex items-center gap-2">
                        <span className="text-sm text-muted-foreground">
                            {branches.length} {branches.length === 1 ? 'branch' : 'branches'}
                        </span>
                        <Button size="sm" className="h-7" onClick={() => setShowCreateDialog(true)}>
                            <Plus className="h-3 w-3 mr-1" />
                            New Branch
                        </Button>
                    </div>
                </div>
            </div>

            <div className="flex-1 overflow-auto">
                <div className="p-4 space-y-4">
                    {/* Current Branch */}
                    {currentBranch && (
                        <div>
                            <h3 className="text-sm font-medium text-muted-foreground mb-2">Current Branch</h3>
                            <div className="border rounded-lg p-4 bg-primary/5 border-primary/20">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <div className="flex items-center gap-2">
                                            <GitBranch className="h-4 w-4 text-primary" />
                                            <span className="font-medium">{currentBranch.name}</span>
                                            <Badge variant="secondary" className="h-5 px-2 text-xs">
                                                <Check className="h-3 w-3 mr-1" />
                                                Current
                                            </Badge>
                                        </div>
                                    </div>
                                    <div className="flex gap-2">
                                        <Button variant="outline" size="sm" disabled>
                                            Switch to
                                        </Button>
                                    </div>
                                </div>
                                
                                {currentBranch.last_commit && (
                                    <div className="mt-3 pt-3 border-t border-primary/10">
                                        <div className="text-sm">
                                            <div className="font-medium mb-1 text-muted-foreground">Latest commit:</div>
                                            <div className="flex items-start justify-between gap-4">
                                                <div className="flex-1 min-w-0">
                                                    <p className="text-sm break-words mb-1">
                                                        {currentBranch.last_commit.message}
                                                    </p>
                                                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                                                        <div className="flex items-center gap-1">
                                                            <User className="h-3 w-3" />
                                                            <span>{currentBranch.last_commit.author.name}</span>
                                                        </div>
                                                        <div className="flex items-center gap-1">
                                                            <Calendar className="h-3 w-3" />
                                                            <span>
                                                                {format(new Date(currentBranch.last_commit.date), 'MMM d, yyyy')}
                                                            </span>
                                                        </div>
                                                        <div className="flex items-center gap-1 font-mono">
                                                            <Hash className="h-3 w-3" />
                                                            <span>{currentBranch.last_commit.hash.substring(0, 7)}</span>
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                )}
                            </div>
                        </div>
                    )}

                    {/* Other Branches */}
                    {otherBranches.length > 0 && (
                        <div>
                            <h3 className="text-sm font-medium text-muted-foreground mb-2">
                                Other Branches ({otherBranches.length})
                            </h3>
                            <div className="space-y-2">
                                {otherBranches.map((branch) => (
                                    <div
                                        key={branch.name}
                                        className="border rounded-lg p-4 hover:bg-muted/50 transition-colors"
                                    >
                                        <div className="flex items-center justify-between">
                                            <div className="flex items-center gap-3">
                                                <div className="flex items-center gap-2">
                                                    <GitBranch className="h-4 w-4 text-muted-foreground" />
                                                    <span className="font-medium">{branch.name}</span>
                                                    {branch.is_remote && (
                                                        <Badge variant="outline" className="h-5 px-2 text-xs">
                                                            Remote
                                                        </Badge>
                                                    )}
                                                </div>
                                            </div>
                                            <div className="flex gap-2">
                                                <Button 
                                                    variant="outline" 
                                                    size="sm"
                                                    onClick={() => handleSwitchBranch(branch.name)}
                                                    disabled={switchBranchMutation.isPending}
                                                >
                                                    {switchBranchMutation.isPending ? (
                                                        <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                                                    ) : null}
                                                    Switch to
                                                </Button>
                                                <Button 
                                                    variant="ghost" 
                                                    size="sm"
                                                    onClick={() => setDeleteBranchName(branch.name)}
                                                    disabled={deleteBranchMutation.isPending}
                                                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                                                >
                                                    <Trash2 className="h-3 w-3" />
                                                </Button>
                                            </div>
                                        </div>
                                        
                                        {branch.last_commit && (
                                            <div className="mt-3 pt-3 border-t">
                                                <div className="text-sm">
                                                    <div className="flex items-start justify-between gap-4">
                                                        <div className="flex-1 min-w-0">
                                                            <p className="text-sm text-muted-foreground break-words mb-1">
                                                                {branch.last_commit.message}
                                                            </p>
                                                            <div className="flex items-center gap-3 text-xs text-muted-foreground">
                                                                <div className="flex items-center gap-1">
                                                                    <User className="h-3 w-3" />
                                                                    <span>{branch.last_commit.author.name}</span>
                                                                </div>
                                                                <div className="flex items-center gap-1">
                                                                    <Calendar className="h-3 w-3" />
                                                                    <span>
                                                                        {format(new Date(branch.last_commit.date), 'MMM d, yyyy')}
                                                                    </span>
                                                                </div>
                                                                <div className="flex items-center gap-1 font-mono">
                                                                    <Hash className="h-3 w-3" />
                                                                    <span>{branch.last_commit.hash.substring(0, 7)}</span>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            </div>
            
            <CreateBranchDialog 
                open={showCreateDialog} 
                onOpenChange={setShowCreateDialog} 
            />
            
            <Dialog open={!!deleteBranchName} onOpenChange={() => setDeleteBranchName(null)}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Branch</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete the branch "{deleteBranchName}"? 
                            This action cannot be undone.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button 
                            variant="outline" 
                            onClick={() => setDeleteBranchName(null)}
                            disabled={deleteBranchMutation.isPending}
                        >
                            Cancel
                        </Button>
                        <Button 
                            variant="destructive" 
                            onClick={handleDeleteBranch}
                            disabled={deleteBranchMutation.isPending}
                        >
                            {deleteBranchMutation.isPending ? (
                                <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                            ) : (
                                <Trash2 className="h-3 w-3 mr-1" />
                            )}
                            Delete Branch
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}