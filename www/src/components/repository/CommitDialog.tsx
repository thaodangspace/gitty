import { useState, useMemo } from 'react';
import { useAtom } from 'jotai';
import { selectedRepositoryAtom } from '@/store/atoms';
import { useRepositoryStatus, triggerRepositoryStatus, useGenerateCommitMessage } from '@/store/queries';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    MobileDialogContent,
} from '@/components/ui/dialog';
import { useIsMobile } from '@/hooks/use-mobile';
import { Drawer } from '@/components/ui/drawer';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { GitCommit, Loader2, CheckCircle2, Sparkles } from 'lucide-react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import type { FileChange } from '@/types/api';

interface CommitDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export default function CommitDialog({ open, onOpenChange }: CommitDialogProps) {
    const [currentRepository] = useAtom(selectedRepositoryAtom);
    const { data: repoStatus } = useRepositoryStatus(currentRepository?.id);
    const [commitTitle, setCommitTitle] = useState('');
    const [commitDetail, setCommitDetail] = useState('');
    const [authorName, setAuthorName] = useState('');
    const [authorEmail, setAuthorEmail] = useState('');
    const isMobile = useIsMobile();

    const queryClient = useQueryClient();

    const generateCommitMessageMutation = useGenerateCommitMessage();

    // Normalize repo status arrays to explicit types with stable identity
    const staged = useMemo(() => (repoStatus?.staged ?? []) as FileChange[], [repoStatus]);

    const createCommitMutation = useMutation({
        mutationFn: async (data: {
            message: string;
            files: string[];
            author?: { name: string; email: string };
        }) => {
            if (!currentRepository) throw new Error('No repository selected');
            return apiClient.createCommit(currentRepository.id, data);
        },
        onSuccess: async () => {
            if (!currentRepository) {
                return;
            }

            queryClient.invalidateQueries({
                queryKey: ['repository-status', currentRepository.id],
            });
            queryClient.invalidateQueries({ queryKey: ['commit-history', currentRepository.id] });
            await triggerRepositoryStatus(queryClient, currentRepository.id);
            setCommitTitle('');
            setCommitDetail('');
            setAuthorName('');
            setAuthorEmail('');
            onOpenChange(false);
        },
    });

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();

        if (!currentRepository || !commitTitle.trim() || !staged.length) {
            return;
        }

        const stagedFiles = staged.map((file) => file.path);
        const author =
            authorName.trim() && authorEmail.trim()
                ? { name: authorName.trim(), email: authorEmail.trim() }
                : undefined;

        // Combine title and detail into a single commit message
        const fullMessage = commitDetail.trim()
            ? `${commitTitle.trim()}\n\n${commitDetail.trim()}`
            : commitTitle.trim();

        createCommitMutation.mutate({
            message: fullMessage,
            files: stagedFiles,
            author,
        });
    };

    const handleOpenChange = (newOpen: boolean) => {
        if (!createCommitMutation.isPending) {
            onOpenChange(newOpen);
            if (!newOpen) {
                setCommitTitle('');
                setCommitDetail('');
                setAuthorName('');
                setAuthorEmail('');
            }
        }
    };

    const handleGenerateMessage = async () => {
        if (!currentRepository || stagedFilesCount === 0) {
            return;
        }

        try {
            const result = await generateCommitMessageMutation.mutateAsync({
                repositoryId: currentRepository.id,
            });
            // Strip "json" prefix if present, then parse the JSON string
            let jsonString = result.message;
            if (jsonString.startsWith('json')) {
                jsonString = jsonString.replace(/^json\n?/, '');
            }
            const parsed = JSON.parse(jsonString) as { message: string; detail: string };
            setCommitTitle(parsed.message);
            setCommitDetail(parsed.detail || '');
        } catch (error) {
            console.error('Failed to generate commit message:', error);
        }
    };

    const stagedFilesCount = staged.length || 0;

    const dialogContent = (
        <>
            <DialogHeader>
                <DialogTitle className="flex items-center gap-2">
                    <GitCommit className={isMobile ? 'h-5 w-5' : 'h-4 w-4'} />
                    Create Commit
                </DialogTitle>
                <DialogDescription>
                    Commit {stagedFilesCount} staged {stagedFilesCount === 1 ? 'file' : 'files'} to{' '}
                    {currentRepository?.current_branch}
                </DialogDescription>
            </DialogHeader>

            {stagedFilesCount === 0 ? (
                <div className="py-6 text-center text-muted-foreground">
                    <CheckCircle2 className="h-8 w-8 mx-auto mb-2 opacity-50" />
                    <p>No staged files to commit</p>
                    <p className="text-sm">Stage some changes first</p>
                </div>
            ) : (
                <form onSubmit={handleSubmit}>
                    <div className="grid gap-4 py-4">
                        <div className="grid gap-2">
                            <label htmlFor="commitTitle" className="text-sm font-medium">
                                Title *
                            </label>
                            <div className="flex gap-2">
                                <Input
                                    id="commitTitle"
                                    value={commitTitle}
                                    onChange={(e) => setCommitTitle(e.target.value)}
                                    placeholder="Add a meaningful commit title..."
                                    disabled={createCommitMutation.isPending}
                                    autoFocus
                                    className="flex-1"
                                />
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="icon"
                                    onClick={handleGenerateMessage}
                                    disabled={
                                        stagedFilesCount === 0 ||
                                        generateCommitMessageMutation.isPending ||
                                        createCommitMutation.isPending
                                    }
                                    title="Generate commit message with AI"
                                >
                                    {generateCommitMessageMutation.isPending ? (
                                        <Loader2 className="h-4 w-4 animate-spin" />
                                    ) : (
                                        <Sparkles className="h-4 w-4" />
                                    )}
                                </Button>
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <label htmlFor="commitDetail" className="text-sm font-medium">
                                Description
                            </label>
                            <Textarea
                                id="commitDetail"
                                value={commitDetail}
                                onChange={(e) => setCommitDetail(e.target.value)}
                                placeholder="Add a detailed description..."
                                disabled={createCommitMutation.isPending}
                                rows={3}
                            />
                        </div>

                        <div className={isMobile ? 'grid gap-4' : 'grid grid-cols-2 gap-4'}>
                            <div className="grid gap-2">
                                <label htmlFor="authorName" className="text-sm font-medium">
                                    Author name
                                </label>
                                <Input
                                    id="authorName"
                                    value={authorName}
                                    onChange={(e) => setAuthorName(e.target.value)}
                                    placeholder="Your name"
                                    disabled={createCommitMutation.isPending}
                                />
                            </div>

                            <div className="grid gap-2">
                                <label htmlFor="authorEmail" className="text-sm font-medium">
                                    Author email
                                </label>
                                <Input
                                    id="authorEmail"
                                    type="email"
                                    value={authorEmail}
                                    onChange={(e) => setAuthorEmail(e.target.value)}
                                    placeholder="your@email.com"
                                    disabled={createCommitMutation.isPending}
                                />
                            </div>
                        </div>

                        {staged.length > 0 && (
                            <div className="grid gap-2">
                                <label className="text-sm font-medium">
                                    Staged files ({staged.length})
                                </label>
                                <div className="max-h-32 overflow-y-auto border rounded-md p-2 bg-muted/50">
                                    {staged.map((file, index) => (
                                        <div key={index} className="text-sm py-1 text-green-600">
                                            {file.status} {file.path}
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {createCommitMutation.isError && (
                            <p className="text-sm text-red-600">
                                Failed to create commit: {createCommitMutation.error.message}
                            </p>
                        )}
                        {generateCommitMessageMutation.isError && (
                            <p className="text-sm text-amber-600">
                                Failed to generate commit message: {generateCommitMessageMutation.error.message}
                            </p>
                        )}
                    </div>

                    <DialogFooter className={isMobile ? 'flex-col gap-2' : ''}>
                        <Button
                            type="button"
                            variant="outline"
                            onClick={() => handleOpenChange(false)}
                            disabled={createCommitMutation.isPending}
                            className={isMobile ? 'w-full' : ''}
                        >
                            Cancel
                        </Button>
                        <Button
                            type="submit"
                            disabled={!commitTitle.trim() || createCommitMutation.isPending}
                            className={isMobile ? 'w-full' : ''}
                        >
                            {createCommitMutation.isPending && (
                                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                            )}
                            Create Commit
                        </Button>
                    </DialogFooter>
                </form>
            )}
        </>
    );

    return isMobile ? (
        <Drawer open={open} onOpenChange={handleOpenChange}>
            <MobileDialogContent>{dialogContent}</MobileDialogContent>
        </Drawer>
    ) : (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[500px]">{dialogContent}</DialogContent>
        </Dialog>
    );
}
