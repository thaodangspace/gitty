import { useAtom } from 'jotai';
import { selectedRepositoryAtom, selectedRepositoryIdAtom, showCreateRepositoryDialogAtom, showChooseRepositoryDialogAtom } from '@/store/atoms';
import { useRepositories } from '@/hooks/api';
import { Button } from '@/components/ui/button';
import { Folder, Plus, GitBranch, Loader2, FolderOpen } from 'lucide-react';
import type { Repository } from '@/types/api';

export default function RepositoryList() {
  const { data: repositories, isLoading, error } = useRepositories();
  const [selectedRepository, setSelectedRepository] = useAtom(selectedRepositoryAtom);
  const [, setSelectedRepositoryId] = useAtom(selectedRepositoryIdAtom);
  const [, setShowCreateDialog] = useAtom(showCreateRepositoryDialogAtom);
  const [, setShowChooseDialog] = useAtom(showChooseRepositoryDialogAtom);

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground p-3">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-xs md:text-sm">Loading repositories...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 p-3">
        Failed to load repositories
      </div>
    );
  }

  const handleRepositorySelect = (repo: Repository) => {
    console.log('RepositoryList - selecting repository:', repo);
    setSelectedRepository(repo);
    setSelectedRepositoryId(repo.id);
  };

  const handleCreateRepository = () => {
    setShowCreateDialog(true);
  };

  const handleChooseRepository = () => {
    setShowChooseDialog(true);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold">Repositories</h3>
        <div className="flex gap-1">
          <Button 
            size="sm" 
            variant="ghost" 
            onClick={handleChooseRepository} 
            title="Import existing repository"
            className="touch-target p-2"
          >
            <FolderOpen className="h-4 w-4" />
          </Button>
          <Button 
            size="sm" 
            variant="ghost" 
            onClick={handleCreateRepository} 
            title="Create new repository"
            className="touch-target p-2"
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      </div>
      
      <div className="space-y-1">
        {repositories?.length === 0 ? (
          <div className="text-xs text-muted-foreground py-3 text-center">
            <p className="mb-2">No repositories found</p>
            <div className="flex flex-col gap-2">
              <Button 
                variant="link" 
                className="text-xs p-0 h-auto touch-target"
                onClick={handleCreateRepository}
              >
                Create your first repository
              </Button>
              <span className="text-xs">or</span>
              <Button 
                variant="link" 
                className="text-xs p-0 h-auto touch-target"
                onClick={handleChooseRepository}
              >
                Import an existing repository
              </Button>
            </div>
          </div>
        ) : (
          repositories?.map((repo) => (
            <Button
              key={repo.id}
              variant={selectedRepository?.id === repo.id ? 'secondary' : 'ghost'}
              className="w-full justify-start text-left h-auto p-3 touch-target"
              onClick={() => handleRepositorySelect(repo)}
            >
              <div className="flex items-start gap-3 w-full">
                <Folder className="h-4 w-4 mt-0.5 shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="font-medium text-sm truncate">
                    {repo.name}
                  </div>
                  <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
                    <GitBranch className="h-3 w-3" />
                    <span className="truncate">
                      {repo.current_branch || 'main'}
                    </span>
                  </div>
                  {repo.is_local && (
                    <div className="text-xs text-blue-600 mt-1">Local</div>
                  )}
                </div>
              </div>
            </Button>
          ))
        )}
      </div>
    </div>
  );
}