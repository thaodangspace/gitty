import { useState } from 'react';
import { useAtom } from 'jotai';
import { selectedRepositoryFromListAtom } from '@/store/atoms';
import { useCreateBranch } from '@/store/queries';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { GitBranch, Loader2 } from 'lucide-react';

interface CreateBranchDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export default function CreateBranchDialog({ open, onOpenChange }: CreateBranchDialogProps) {
    const [currentRepository] = useAtom(selectedRepositoryFromListAtom);
    const [branchName, setBranchName] = useState('');
    const createBranchMutation = useCreateBranch();

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        
        if (!currentRepository || !branchName.trim()) {
            return;
        }

        createBranchMutation.mutate(
            {
                repositoryId: currentRepository.id,
                branchName: branchName.trim(),
            },
            {
                onSuccess: () => {
                    setBranchName('');
                    onOpenChange(false);
                },
                onError: (error) => {
                    console.error('Failed to create branch:', error);
                },
            }
        );
    };

    const handleOpenChange = (newOpen: boolean) => {
        if (!createBranchMutation.isPending) {
            onOpenChange(newOpen);
            if (!newOpen) {
                setBranchName('');
            }
        }
    };

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <GitBranch className="h-4 w-4" />
                        Create New Branch
                    </DialogTitle>
                    <DialogDescription>
                        Create a new branch from the current branch ({currentRepository?.current_branch})
                    </DialogDescription>
                </DialogHeader>
                <form onSubmit={handleSubmit}>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <label htmlFor="branchName" className="text-sm font-medium">
                                Branch name
                            </label>
                            <Input
                                id="branchName"
                                value={branchName}
                                onChange={(e) => setBranchName(e.target.value)}
                                placeholder="feature/new-feature"
                                disabled={createBranchMutation.isPending}
                                autoFocus
                            />
                            {createBranchMutation.isError && (
                                <p className="text-sm text-red-600">
                                    Failed to create branch: {createBranchMutation.error.message}
                                </p>
                            )}
                        </div>
                    </div>
                    <DialogFooter>
                        <Button
                            type="button"
                            variant="outline"
                            onClick={() => handleOpenChange(false)}
                            disabled={createBranchMutation.isPending}
                        >
                            Cancel
                        </Button>
                        <Button
                            type="submit"
                            disabled={!branchName.trim() || createBranchMutation.isPending}
                        >
                            {createBranchMutation.isPending && (
                                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                            )}
                            Create Branch
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
}