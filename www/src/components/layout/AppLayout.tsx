import { useAtom } from 'jotai';
import { sidebarOpenAtom, sidebarWidthAtom, globalErrorAtom, globalLoadingAtom } from '@/store/atoms';
import Header from './Header';
import Sidebar from './Sidebar';
import MainContent from './MainContent';
import StatusBar from './StatusBar';
import ChooseRepositoryDialog from '@/components/repository/ChooseRepositoryDialog';
import { useEffect, useState } from 'react';
import { useRepositories } from '@/hooks/api';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2 } from 'lucide-react';
import { Drawer, DrawerContent } from '@/components/ui/drawer';

export default function AppLayout() {
  const [sidebarOpen, setSidebarOpen] = useAtom(sidebarOpenAtom);
  const [sidebarWidth] = useAtom(sidebarWidthAtom);
  const [globalError, setGlobalError] = useAtom(globalErrorAtom);
  const [globalLoading] = useAtom(globalLoadingAtom);
  const [isMobile, setIsMobile] = useState(false);
  
  // Initialize repositories on app load
  const { error: repoError } = useRepositories();

  useEffect(() => {
    if (repoError) {
      setGlobalError(repoError.message || 'Failed to load repositories');
    }
  }, [repoError, setGlobalError]);

  // Track mobile state
  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768);
    };
    
    checkMobile();
    window.addEventListener('resize', checkMobile);
    
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  // Handle mobile sidebar events
  useEffect(() => {
    const handleCloseSidebar = () => {
      if (isMobile) {
        setSidebarOpen(false);
      }
    };

    const handleResize = () => {
      // Auto-close sidebar on mobile when screen gets smaller
      if (isMobile && sidebarOpen) {
        setSidebarOpen(false);
      }
    };

    window.addEventListener('closeSidebar', handleCloseSidebar);
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('closeSidebar', handleCloseSidebar);
      window.removeEventListener('resize', handleResize);
    };
  }, [sidebarOpen, setSidebarOpen, isMobile]);

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
        
        {/* Desktop Sidebar */}
        <div 
          className={`
            hidden md:flex
            ${sidebarOpen ? 'block' : 'hidden'}
            bg-background border-r
          `}
          style={{ width: sidebarWidth }}
        >
          <Sidebar />
        </div>

        {/* Mobile Drawer */}
        {isMobile && (
          <Drawer open={sidebarOpen} onOpenChange={setSidebarOpen} direction="bottom">
            <DrawerContent className="max-h-[85vh] rounded-t-xl">
              <div className="mx-auto w-12 h-1.5 flex-shrink-0 rounded-full bg-muted-foreground/20 mb-4 mt-2" />
              <Sidebar />
            </DrawerContent>
          </Drawer>
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