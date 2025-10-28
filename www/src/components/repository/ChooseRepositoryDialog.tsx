import { useState } from 'react';
import { useAtom } from 'jotai';
import {
    showChooseRepositoryDialogAtom,
    currentDirectoryAtom,
    selectedRepositoryAtom,
    selectedRepositoryIdAtom,
} from '@/store/atoms';
import {
    useDirectoryBrowse,
    useVolumeRoots,
    useImportRepository,
    useRepositories,
} from '@/hooks/api';
import { useIsMobile } from '@/hooks/use-mobile';
import { Button } from '@/components/ui/button';
import { Folder, Home, HardDrive, ChevronUp, Loader2, FolderGit2, X } from 'lucide-react';
import type { DirectoryEntry, Repository } from '@/types/api';

export default function ChooseRepositoryDialog() {
    const [showDialog, setShowDialog] = useAtom(showChooseRepositoryDialogAtom);
    const [currentDirectory, setCurrentDirectory] = useAtom(currentDirectoryAtom);
    const [, setSelectedRepository] = useAtom(selectedRepositoryAtom);
    const [, setSelectedRepositoryId] = useAtom(selectedRepositoryIdAtom);
    const [selectedRepo, setSelectedRepo] = useState<DirectoryEntry | null>(null);
    const [pathInput, setPathInput] = useState('');
    const [isPathReadonly, setIsPathReadonly] = useState(true);
    const isMobile = useIsMobile();

    const { data: repositories } = useRepositories();
    const { data: directoryListing, isLoading, error } = useDirectoryBrowse(currentDirectory);
    const { data: volumeRoots } = useVolumeRoots();
    const importRepository = useImportRepository();

    const handleRepositorySelect = (repo: Repository) => {
        setSelectedRepository(repo);
        setSelectedRepositoryId(repo.id);
        setShowDialog(false);
        setSelectedRepo(null);
        setPathInput('');
        setCurrentDirectory('');
    };

    const handleDirectoryClick = (entry: DirectoryEntry) => {
        if (entry.is_directory) {
            if (entry.is_git_repo) {
                setSelectedRepo(entry);
                setPathInput(entry.path);
                setIsPathReadonly(true);
            } else {
                setCurrentDirectory(entry.path);
                setPathInput(entry.path);
                setIsPathReadonly(true);
            }
        }
    };

    const handleParentDirectory = () => {
        if (directoryListing?.parent_path) {
            setCurrentDirectory(directoryListing.parent_path);
            setPathInput(directoryListing.parent_path);
            setIsPathReadonly(true);
        }
    };

    const handleRootClick = (root: DirectoryEntry) => {
        setCurrentDirectory(root.path);
        setPathInput(root.path);
        setSelectedRepo(null);
        setIsPathReadonly(true);
    };

    const handleImport = async () => {
        const trimmedPath = pathInput.trim() || selectedRepo?.path;

        if (!trimmedPath) {
            return;
        }

        const fallbackName = selectedRepo?.name || getPathBasename(trimmedPath);

        try {
            const payload = fallbackName
                ? { path: trimmedPath, name: fallbackName }
                : { path: trimmedPath };

            const importedRepo = await importRepository.mutateAsync(payload);

            if (importedRepo) {
                setSelectedRepository(importedRepo);
                setSelectedRepositoryId(importedRepo.id);
            }

            setShowDialog(false);
            setSelectedRepo(null);
            setPathInput('');
            setCurrentDirectory('');
        } catch (error) {
            console.error('Failed to import repository:', error);
        }
    };

    const handleCancel = () => {
        setShowDialog(false);
        setSelectedRepo(null);
        setPathInput('');
        setCurrentDirectory('');
    };

    const handlePathSubmit = () => {
        const trimmedPath = pathInput.trim();

        if (!trimmedPath) {
            return;
        }

        // Check if the entered path matches a git repo in current listing
        const matchingEntry = directoryListing?.entries.find(
            (entry) => entry.path === trimmedPath && entry.is_git_repo
        );

        if (matchingEntry) {
            // If it's a git repo, select it for import
            setSelectedRepo(matchingEntry);
            setIsPathReadonly(true);
        } else {
            // Otherwise, browse to it as a directory
            setCurrentDirectory(trimmedPath);
            setSelectedRepo(null);
            setIsPathReadonly(true);
        }
    };

    const handleManualPathChange = (value: string) => {
        setPathInput(value);
        setSelectedRepo(null);
        setIsPathReadonly(false);
    };

    if (!showDialog) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center md:p-4">
            {/* Backdrop */}
            <div className="fixed inset-0 bg-black/50" onClick={handleCancel} />

            {/* Dialog - Bottom drawer on mobile, centered modal on desktop */}
            <div
                className={`
        relative bg-background border shadow-lg w-full flex flex-col
        ${
            isMobile
                ? 'fixed bottom-0 left-0 right-0 h-[85vh] max-h-[85vh] overflow-hidden rounded-t-xl animate-in slide-in-from-bottom-full duration-300'
                : 'rounded-lg max-w-4xl h-[700px] animate-in fade-in-0 zoom-in-95 duration-200'
        }
      `}
            >
                {/* Header */}
                <div className="p-4 md:p-6 border-b">
                    {/* Mobile drag handle */}
                    {isMobile && (
                        <div className="flex justify-center mb-3">
                            <div className="w-12 h-1.5 bg-muted-foreground/30 rounded-full" />
                        </div>
                    )}
                    <div className="flex items-center justify-between">
                        <div className="min-w-0 flex-1">
                            <h2 className="text-lg font-semibold">Select or Import Repository</h2>
                            <p className="text-sm text-muted-foreground truncate">
                                Choose an existing repository or browse to import a new one
                            </p>
                        </div>
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={handleCancel}
                            className="touch-target p-2 ml-2"
                        >
                            <X className="h-4 w-4" />
                        </Button>
                    </div>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-hidden flex flex-col">
                    <div className="flex flex-col md:flex-row gap-4 flex-1 min-h-0 p-4 md:p-6">
                        {/* Quick Access sidebar - repositories and volume roots */}
                        <div className="w-full md:w-48 border-b md:border-b-0 md:border-r pb-4 md:pb-0 md:pr-4 flex flex-col gap-4 md:overflow-y-auto">
                            {/* Your Repositories */}
                            {repositories && repositories.length > 0 && (
                                <div className="flex-shrink-0">
                                    <h4 className="text-sm font-medium mb-2">Your Repositories</h4>
                                    <div className="flex md:flex-col gap-1 overflow-x-auto md:overflow-x-visible md:max-h-[200px] md:overflow-y-auto">
                                        {repositories.map((repo) => (
                                            <Button
                                                key={repo.id}
                                                variant="ghost"
                                                size="sm"
                                                className="flex-shrink-0 md:w-full md:justify-start touch-target h-auto py-2"
                                                onClick={() => handleRepositorySelect(repo)}
                                            >
                                                <div className="flex items-center gap-2 w-full min-w-0">
                                                    <Folder className="h-4 w-4 flex-shrink-0" />
                                                    <div className="min-w-0 flex-1 text-left">
                                                        <div className="text-sm truncate">
                                                            {repo.name}
                                                        </div>
                                                    </div>
                                                </div>
                                            </Button>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* Import New Repository */}
                            <div className="flex-shrink-0">
                                <h4 className="text-sm font-medium mb-2">Quick Access</h4>
                                <div className="flex md:flex-col gap-2 md:gap-1 overflow-x-auto md:overflow-x-visible">
                                    {volumeRoots?.roots.map((root) => (
                                        <Button
                                            key={root.path}
                                            variant="ghost"
                                            size="sm"
                                            className="flex-shrink-0 md:w-full md:justify-start touch-target"
                                            onClick={() => handleRootClick(root)}
                                        >
                                            {root.name === '/' ? (
                                                <HardDrive className="h-4 w-4 mr-2" />
                                            ) : root.name === 'Home' ? (
                                                <Home className="h-4 w-4 mr-2" />
                                            ) : (
                                                <HardDrive className="h-4 w-4 mr-2" />
                                            )}
                                            <span className="truncate">{root.name}</span>
                                        </Button>
                                    ))}
                                </div>
                            </div>
                        </div>

                        {/* Directory browser */}
                        <div className="flex-1 flex flex-col min-w-0 min-h-0">
                            {/* Navigation */}
                            <div className="flex items-center gap-2 mb-3 flex-shrink-0">
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={handleParentDirectory}
                                    disabled={!directoryListing?.can_go_up}
                                    className="touch-target"
                                >
                                    <ChevronUp className="h-4 w-4 mr-1" />
                                    Up
                                </Button>
                            </div>

                            {/* Directory listing */}
                            <div className="flex-1 border rounded overflow-y-auto min-h-0">
                                {isLoading ? (
                                    <div className="flex items-center justify-center p-8">
                                        <Loader2 className="h-6 w-6 animate-spin" />
                                    </div>
                                ) : error ? (
                                    <div className="p-4 text-red-500 text-sm">
                                        Failed to load directory
                                    </div>
                                ) : (
                                    <div className="p-2">
                                        {directoryListing?.entries
                                            .filter(
                                                (entry) => entry.is_directory && !entry.is_hidden
                                            )
                                            .map((entry) => (
                                                <Button
                                                    key={entry.path}
                                                    variant={
                                                        selectedRepo?.path === entry.path
                                                            ? 'secondary'
                                                            : 'ghost'
                                                    }
                                                    className="w-full justify-start mb-1 h-auto p-3 touch-target"
                                                    onClick={() => handleDirectoryClick(entry)}
                                                >
                                                    <div className="flex items-center gap-3 w-full">
                                                        {entry.is_git_repo ? (
                                                            <FolderGit2 className="h-4 w-4 text-green-600 flex-shrink-0" />
                                                        ) : (
                                                            <Folder className="h-4 w-4 flex-shrink-0" />
                                                        )}
                                                        <span className="truncate text-left flex-1">
                                                            {entry.name}
                                                        </span>
                                                        {entry.is_git_repo && (
                                                            <span className="text-xs text-green-600 font-medium flex-shrink-0">
                                                                Git Repository
                                                            </span>
                                                        )}
                                                    </div>
                                                </Button>
                                            ))}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>

                    {/* Selected repository details */}
                    <div className="space-y-4 border-t p-4 md:p-6 flex-shrink-0">
                        <div>
                            <label className="text-sm font-medium">Repository Path</label>
                            <div className="mt-1">
                                <input
                                    type="text"
                                    value={pathInput || directoryListing?.current_path || ''}
                                    onChange={(e) => handleManualPathChange(e.target.value)}
                                    onFocus={() => setIsPathReadonly(false)}
                                    onBlur={() => {
                                        if (!pathInput) {
                                            setIsPathReadonly(true);
                                        }
                                    }}
                                    onKeyDown={(event) => {
                                        if (event.key === 'Enter') {
                                            event.preventDefault();
                                            handlePathSubmit();
                                        }
                                    }}
                                    readOnly={isPathReadonly}
                                    placeholder="Select a location or type a path"
                                    className={`w-full px-3 py-2 border border-input rounded-md text-sm h-10 transition-colors ${
                                        isPathReadonly
                                            ? 'bg-muted/50 cursor-pointer'
                                            : 'bg-background'
                                    }`}
                                />
                            </div>
                            <p className="text-xs text-muted-foreground mt-1">
                                Click to edit the path manually, or use the browser above to
                                navigate. Press Enter to browse to a directory or select a git
                                repository.
                            </p>
                        </div>

                        {selectedRepo && (
                            <div className="p-3 bg-muted/30 rounded">
                                <div className="flex items-center gap-2">
                                    <FolderGit2 className="h-4 w-4 text-green-600 flex-shrink-0" />
                                    <span className="font-medium">Selected Repository:</span>
                                    <span className="text-sm text-muted-foreground truncate">
                                        {selectedRepo.path}
                                    </span>
                                </div>
                            </div>
                        )}
                    </div>
                </div>

                {/* Footer */}
                <div className="flex justify-end gap-2 p-4 md:p-6 border-t">
                    <Button variant="outline" onClick={handleCancel} className="touch-target">
                        Cancel
                    </Button>
                    <Button
                        onClick={handleImport}
                        disabled={
                            (!pathInput.trim() && !selectedRepo?.path) || importRepository.isPending
                        }
                        className="touch-target"
                    >
                        {importRepository.isPending ? (
                            <>
                                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                                Importing...
                            </>
                        ) : (
                            'Import Repository'
                        )}
                    </Button>
                </div>
            </div>
        </div>
    );
}

const getPathBasename = (path: string) => {
    const sanitizedPath = path.replace(/[\\/]+$/, '');

    if (!sanitizedPath) {
        return '';
    }

    const segments = sanitizedPath.split(/[\\/]/).filter(Boolean);
    return segments[segments.length - 1] || '';
};
