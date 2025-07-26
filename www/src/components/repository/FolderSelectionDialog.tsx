import { useState } from 'react';
import { useAtom } from 'jotai';
import { showFolderSelectionDialogAtom, currentDirectoryAtom } from '@/store/atoms';
import { useDirectoryBrowse, useVolumeRoots } from '@/hooks/api';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { 
  Folder, 
  FolderOpen, 
  Home, 
  HardDrive, 
  ChevronUp, 
  Loader2,
  GitBranch 
} from 'lucide-react';
import type { DirectoryEntry } from '@/types/api';

interface FolderSelectionDialogProps {
  onSelectPath: (path: string) => void;
  title?: string;
  description?: string;
}

export default function FolderSelectionDialog({ 
  onSelectPath, 
  title = "Select Folder",
  description = "Choose a folder for your repository"
}: FolderSelectionDialogProps) {
  const [showDialog, setShowDialog] = useAtom(showFolderSelectionDialogAtom);
  const [currentDirectory, setCurrentDirectory] = useAtom(currentDirectoryAtom);
  const [selectedPath, setSelectedPath] = useState<string>('');

  const { data: directoryListing, isLoading, error } = useDirectoryBrowse(currentDirectory);
  const { data: volumeRoots } = useVolumeRoots();

  const handleDirectoryClick = (entry: DirectoryEntry) => {
    if (entry.is_directory) {
      setCurrentDirectory(entry.path);
      setSelectedPath(entry.path);
    }
  };

  const handleParentDirectory = () => {
    if (directoryListing?.parent_path) {
      setCurrentDirectory(directoryListing.parent_path);
    }
  };

  const handleRootClick = (root: DirectoryEntry) => {
    setCurrentDirectory(root.path);
    setSelectedPath(root.path);
  };

  const handleSelect = () => {
    if (selectedPath) {
      onSelectPath(selectedPath);
      setShowDialog(false);
      setSelectedPath('');
      setCurrentDirectory('');
    }
  };

  const handleCancel = () => {
    setShowDialog(false);
    setSelectedPath('');
    setCurrentDirectory('');
  };

  return (
    <Dialog open={showDialog} onOpenChange={setShowDialog}>
      <DialogContent className="max-w-2xl h-[600px] flex flex-col">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && (
            <p className="text-sm text-muted-foreground">{description}</p>
          )}
        </DialogHeader>

        <div className="flex-1 flex gap-4 min-h-0">
          {/* Volume roots sidebar */}
          <div className="w-48 border-r pr-4">
            <h4 className="text-sm font-medium mb-2">Quick Access</h4>
            <div className="space-y-1">
              {volumeRoots?.roots.map((root) => (
                <Button
                  key={root.path}
                  variant="ghost"
                  size="sm"
                  className="w-full justify-start"
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
            <div className="mb-2">
              <div className="flex items-center gap-2 p-2 bg-muted/50 rounded text-sm">
                <Folder className="h-4 w-4" />
                <span className="truncate">{directoryListing?.current_path || 'Select a location'}</span>
              </div>
            </div>

            {/* Navigation */}
            <div className="flex items-center gap-2 mb-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleParentDirectory}
                disabled={!directoryListing?.can_go_up}
              >
                <ChevronUp className="h-4 w-4 mr-1" />
                Up
              </Button>
            </div>

            {/* Directory listing */}
            <ScrollArea className="flex-1 border rounded">
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
                        variant={selectedPath === entry.path ? 'secondary' : 'ghost'}
                        className="w-full justify-start mb-1 h-auto p-2"
                        onClick={() => handleDirectoryClick(entry)}
                        onDoubleClick={() => handleDirectoryClick(entry)}
                      >
                        <div className="flex items-center gap-2 w-full">
                          {entry.is_git_repo ? (
                            <div className="relative">
                              <Folder className="h-4 w-4" />
                              <GitBranch className="h-2 w-2 absolute -top-1 -right-1 text-green-600" />
                            </div>
                          ) : (
                            <Folder className="h-4 w-4" />
                          )}
                          <span className="truncate text-left flex-1">
                            {entry.name}
                          </span>
                          {entry.is_git_repo && (
                            <span className="text-xs text-green-600">Git</span>
                          )}
                        </div>
                      </Button>
                    ))}
                </div>
              )}
            </ScrollArea>
          </div>
        </div>

        {/* Selected path display */}
        {selectedPath && (
          <div className="p-2 bg-muted/30 rounded text-sm">
            <strong>Selected:</strong> {selectedPath}
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel}>
            Cancel
          </Button>
          <Button onClick={handleSelect} disabled={!selectedPath}>
            Select Folder
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}