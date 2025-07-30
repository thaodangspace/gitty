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
  const [sidebarOpen, setSidebarOpen] = useAtom(sidebarOpenAtom);
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

  // Handle mobile sidebar events
  useEffect(() => {
    const handleCloseSidebar = () => {
      if (window.innerWidth < 768) {
        setSidebarOpen(false);
      }
    };

    const handleResize = () => {
      // Auto-close sidebar on mobile when screen gets smaller
      if (window.innerWidth < 768 && sidebarOpen) {
        setSidebarOpen(false);
      }
    };

    window.addEventListener('closeSidebar', handleCloseSidebar);
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('closeSidebar', handleCloseSidebar);
      window.removeEventListener('resize', handleResize);
    };
  }, [sidebarOpen, setSidebarOpen]);

  // Close sidebar on mobile when clicking outside
  const handleOverlayClick = () => {
    if (window.innerWidth < 768) {
      setSidebarOpen(false);
    }
  };

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
      <div className="flex-1 flex overflow-hidden relative">
        {/* Mobile overlay for sidebar */}
        {sidebarOpen && (
          <div 
            className="fixed inset-0 bg-black/50 z-40 md:hidden"
            onClick={handleOverlayClick}
          />
        )}
        
        {/* Sidebar - responsive positioning */}
        {sidebarOpen && (
          <div 
            className={`
              fixed md:relative inset-y-0 left-0 z-50
              bg-background border-r shadow-lg md:shadow-none
              w-80 md:w-auto
              transform transition-transform duration-300 ease-in-out
              ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}
              md:translate-x-0
            `}
            style={{ width: window.innerWidth >= 768 ? sidebarWidth : '320px' }}
          >
            <Sidebar />
          </div>
        )}
        
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