import { useAtom } from 'jotai';
import { globalErrorAtom, globalLoadingAtom, selectedRepositoryAtom } from '@/store/atoms';
import Header from './Header';
import MainContent from './MainContent';
import StatusBar from './StatusBar';
import { useEffect } from 'react';
import { useRepositories, useRepository } from '@/hooks/api';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2 } from 'lucide-react';
import { useGlobalVimMode } from '@/hooks/use-vim-navigation';
import { useParams } from 'react-router-dom';

export default function AppLayout() {
    const { repoId } = useParams<{ repoId: string }>();
    const [globalError, setGlobalError] = useAtom(globalErrorAtom);
    const [globalLoading] = useAtom(globalLoadingAtom);
    const [, setSelectedRepository] = useAtom(selectedRepositoryAtom);

    const { error: repoListError } = useRepositories();
    const validRepoId = repoId ?? '';
    const { data: repository, isLoading, error } = useRepository(validRepoId);

    useGlobalVimMode();

    useEffect(() => {
        if (repoListError) {
            setGlobalError(repoListError.message || 'Failed to load repositories');
        }
    }, [repoListError, setGlobalError]);

    useEffect(() => {
        if (repository) {
            setSelectedRepository(repository);
        }
    }, [repository, setSelectedRepository]);

    if (isLoading) {
        return (
            <div className="h-screen flex items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin" />
            </div>
        );
    }

    if (error || !repository) {
        return (
            <div className="h-screen flex items-center justify-center p-4">
                <Alert className="max-w-md">
                    <AlertDescription>
                        {error?.message || 'Repository not found'}
                    </AlertDescription>
                </Alert>
            </div>
        );
    }

    if (globalError) {
        return (
            <div className="h-screen flex items-center justify-center p-4">
                <Alert className="max-w-md">
                    <AlertDescription>{globalError}</AlertDescription>
                </Alert>
            </div>
        );
    }

    return (
        <div className="h-screen flex flex-col bg-background">
            <Header />
            <div className="flex-1 flex overflow-hidden relative">
                <div className="flex-1 flex flex-col min-w-0 relative">
                    {globalLoading && (
                        <div className="flex items-center justify-center p-4 border-b">
                            <Loader2 className="h-4 w-4 animate-spin mr-2" />
                            Loading...
                        </div>
                    )}
                    <MainContent />
                </div>
            </div>
            <StatusBar />
        </div>
    );
}
