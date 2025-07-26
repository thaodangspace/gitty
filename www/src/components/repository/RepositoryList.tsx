import { useAtom } from 'jotai';
import { selectedRepositoryAtom, selectedRepositoryIdAtom, showCreateRepositoryDialogAtom } from '@/store/atoms';
import { useRepositories } from '@/hooks/api';
import { Button } from '@/components/ui/button';
import { Folder, Plus, GitBranch, Loader2 } from 'lucide-react';
import type { Repository } from '@/types/api';

export default function RepositoryList() {
  const { data: repositories, isLoading, error } = useRepositories();
  const [selectedRepository, setSelectedRepository] = useAtom(selectedRepositoryAtom);
  const [, setSelectedRepositoryId] = useAtom(selectedRepositoryIdAtom);
  const [, setShowCreateDialog] = useAtom(showCreateRepositoryDialogAtom);

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground p-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        Loading repositories...
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 p-2">
        Failed to load repositories
      </div>
    );
  }

  const handleRepositorySelect = (repo: Repository) => {
    setSelectedRepository(repo);
    setSelectedRepositoryId(repo.id);
  };

  const handleCreateRepository = () => {
    setShowCreateDialog(true);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-semibold">Repositories</h3>
        <Button size="sm" variant="ghost" onClick={handleCreateRepository}>
          <Plus className="h-4 w-4" />
        </Button>
      </div>
      
      <div className="space-y-1">
        {repositories?.length === 0 ? (
          <div className="text-xs text-muted-foreground py-2 text-center">
            No repositories found
            <br />
            <Button 
              variant="link" 
              className="text-xs p-0 h-auto"
              onClick={handleCreateRepository}
            >
              Add your first repository
            </Button>
          </div>
        ) : (
          repositories?.map((repo) => (
            <Button
              key={repo.id}
              variant={selectedRepository?.id === repo.id ? 'secondary' : 'ghost'}
              className="w-full justify-start text-left h-auto p-2"
              onClick={() => handleRepositorySelect(repo)}
            >
              <div className="flex items-start gap-2 w-full">
                <Folder className="h-4 w-4 mt-0.5 shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="font-medium text-sm truncate">
                    {repo.name}
                  </div>
                  <div className="flex items-center gap-1 text-xs text-muted-foreground">
                    <GitBranch className="h-3 w-3" />
                    <span className="truncate">
                      {repo.current_branch || 'main'}
                    </span>
                  </div>
                  {repo.is_local && (
                    <div className="text-xs text-blue-600">Local</div>
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