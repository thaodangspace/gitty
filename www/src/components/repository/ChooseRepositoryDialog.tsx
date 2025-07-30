import { useState } from 'react';
import { useAtom } from 'jotai';
import { showChooseRepositoryDialogAtom, currentDirectoryAtom } from '@/store/atoms';
import { useDirectoryBrowse, useVolumeRoots, useImportRepository } from '@/hooks/api';
import { Button } from '@/components/ui/button';
import { 
  Folder, 
  Home, 
  HardDrive, 
  ChevronUp, 
  Loader2,
  FolderGit2,
  X
} from 'lucide-react';
import type { DirectoryEntry } from '@/types/api';

export default function ChooseRepositoryDialog() {
  const [showDialog, setShowDialog] = useAtom(showChooseRepositoryDialogAtom);
  const [currentDirectory, setCurrentDirectory] = useAtom(currentDirectoryAtom);
  const [selectedRepo, setSelectedRepo] = useState<DirectoryEntry | null>(null);
  const [customName, setCustomName] = useState('');

  const { data: directoryListing, isLoading, error } = useDirectoryBrowse(currentDirectory);
  const { data: volumeRoots } = useVolumeRoots();
  const importRepository = useImportRepository();

  const handleDirectoryClick = (entry: DirectoryEntry) => {
    if (entry.is_directory) {
      if (entry.is_git_repo) {
        setSelectedRepo(entry);
        setCustomName(entry.name);
      } else {
        setCurrentDirectory(entry.path);
      }
    }
  };

  const handleParentDirectory = () => {
    if (directoryListing?.parent_path) {
      setCurrentDirectory(directoryListing.parent_path);
    }
  };

  const handleRootClick = (root: DirectoryEntry) => {
    setCurrentDirectory(root.path);
    setSelectedRepo(null);
    setCustomName('');
  };

  const handleImport = async () => {
    if (selectedRepo) {
      try {
        await importRepository.mutateAsync({
          path: selectedRepo.path,
          name: customName.trim() || selectedRepo.name,
        });
        setShowDialog(false);
        setSelectedRepo(null);
        setCustomName('');
        setCurrentDirectory('');
      } catch (error) {
        console.error('Failed to import repository:', error);
      }
    }
  };

  const handleCancel = () => {
    setShowDialog(false);
    setSelectedRepo(null);
    setCustomName('');
    setCurrentDirectory('');
  };

  if (!showDialog) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-2 md:p-4">
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/50" onClick={handleCancel} />
      
      {/* Dialog */}
      <div className="relative bg-background border rounded-lg shadow-lg w-full max-w-4xl h-[90vh] md:h-[700px] flex flex-col">
        {/* Header */}
        <div className="p-4 md:p-6 border-b">
          <div className="flex items-center justify-between">
            <div className="min-w-0 flex-1">
              <h2 className="text-lg font-semibold">Choose Existing Repository</h2>
              <p className="text-sm text-muted-foreground truncate">
                Browse and select an existing Git repository to import
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
        <div className="flex-1 flex flex-col md:flex-row gap-4 min-h-0 p-4 md:p-6">
          {/* Volume roots sidebar - responsive */}
          <div className="w-full md:w-48 border-b md:border-b-0 md:border-r pb-4 md:pb-0 md:pr-4">
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

          {/* Directory browser */}
          <div className="flex-1 flex flex-col min-w-0">
            {/* Current path */}
            <div className="mb-3">
              <div className="flex items-center gap-2 p-3 bg-muted/50 rounded text-sm">
                <Folder className="h-4 w-4 flex-shrink-0" />
                <span className="truncate">{directoryListing?.current_path || 'Select a location'}</span>
              </div>
            </div>

            {/* Navigation */}
            <div className="flex items-center gap-2 mb-3">
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
            <div className="flex-1 border rounded overflow-auto">
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
                    .filter(entry => entry.is_directory && !entry.is_hidden)
                    .map((entry) => (
                      <Button
                        key={entry.path}
                        variant={selectedRepo?.path === entry.path ? 'secondary' : 'ghost'}
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
                            <span className="text-xs text-green-600 font-medium flex-shrink-0">Git Repository</span>
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
        {selectedRepo && (
          <div className="space-y-3 border-t p-4 md:p-6">
            <div className="p-3 bg-muted/30 rounded">
              <div className="flex items-center gap-2 mb-2">
                <FolderGit2 className="h-4 w-4 text-green-600 flex-shrink-0" />
                <span className="font-medium">Selected Repository:</span>
                <span className="text-sm text-muted-foreground truncate">{selectedRepo.path}</span>
              </div>
            </div>
            
            <div>
              <label className="text-sm font-medium">Repository Name</label>
              <input
                type="text"
                value={customName}
                onChange={(e) => setCustomName(e.target.value)}
                placeholder="Enter a name for this repository"
                className="mt-1 w-full px-3 py-2 border border-input rounded-md bg-background text-sm h-10"
              />
              <p className="text-xs text-muted-foreground mt-1">
                Leave empty to use the folder name: {selectedRepo.name}
              </p>
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="flex justify-end gap-2 p-4 md:p-6 border-t">
          <Button 
            variant="outline" 
            onClick={handleCancel}
            className="touch-target"
          >
            Cancel
          </Button>
          <Button 
            onClick={handleImport} 
            disabled={!selectedRepo || importRepository.isPending}
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