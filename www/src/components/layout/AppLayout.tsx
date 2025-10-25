import { useAtom } from 'jotai';
import { globalErrorAtom, globalLoadingAtom } from '@/store/atoms';
import Header from './Header';
import MainContent from './MainContent';
import StatusBar from './StatusBar';
import ChooseRepositoryDialog from '@/components/repository/ChooseRepositoryDialog';
import { useEffect } from 'react';
import { useRepositories } from '@/hooks/api';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2 } from 'lucide-react';
import { useGlobalVimMode } from '@/hooks/use-vim-navigation';

export default function AppLayout() {
    const [globalError, setGlobalError] = useAtom(globalErrorAtom);
    const [globalLoading] = useAtom(globalLoadingAtom);

    const { error: repoError } = useRepositories();

    // Initialize global vim mode
    useGlobalVimMode();

    useEffect(() => {
        if (repoError) {
            setGlobalError(repoError.message || 'Failed to load repositories');
        }
    }, [repoError, setGlobalError]);

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
                {/* Main content area */}
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
            <ChooseRepositoryDialog />
        </div>
    );
}
