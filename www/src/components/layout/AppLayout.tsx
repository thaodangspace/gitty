import { useAtom } from 'jotai';
import { sidebarOpenAtom, sidebarWidthAtom, globalErrorAtom, globalLoadingAtom } from '@/store/atoms';
import Header from './Header';
import Sidebar from './Sidebar';
import MainContent from './MainContent';
import StatusBar from './StatusBar';
import ChooseRepositoryDialog from '@/components/repository/ChooseRepositoryDialog';
import { useEffect } from 'react';
import { useRepositories } from '@/hooks/api';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2 } from 'lucide-react';

export default function AppLayout() {
  const [sidebarOpen] = useAtom(sidebarOpenAtom);
  const [sidebarWidth] = useAtom(sidebarWidthAtom);
  const [globalError, setGlobalError] = useAtom(globalErrorAtom);
  const [globalLoading] = useAtom(globalLoadingAtom);
  
  // Initialize repositories on app load
  const { error: repoError } = useRepositories();

  useEffect(() => {
    if (repoError) {
      setGlobalError(repoError.message || 'Failed to load repositories');
    }
  }, [repoError, setGlobalError]);

  if (globalError) {
    return (
      <div className="h-screen flex items-center justify-center p-4">
        <Alert className="max-w-md">
          <AlertDescription>
            {globalError}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col bg-background">
      <Header />
      <div className="flex-1 flex overflow-hidden">
        {sidebarOpen && (
          <div 
            className="bg-muted/30 border-r flex-shrink-0" 
            style={{ width: sidebarWidth }}
          >
            <Sidebar />
          </div>
        )}
        <div className="flex-1 flex flex-col min-w-0">
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