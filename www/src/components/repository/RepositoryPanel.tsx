import { useAtom } from 'jotai';
import { activeViewAtom, selectedRepositoryAtom } from '@/store/atoms';
import { useState } from 'react';
import { useIsMobile } from '@/hooks/use-mobile';
import { Button } from '@/components/ui/button';
import { Drawer, DrawerContent, DrawerHeader, DrawerTitle } from '@/components/ui/drawer';
import { FolderTree, X } from 'lucide-react';
import FileTreeBrowser from '../file/FileTreeBrowser';
import FileViewer from '../file/FileViewer';
import CommitHistory from './CommitHistory';
import CommitTree from './CommitTree';
import BranchList from './BranchList';
import WorkingDirectoryChanges from './WorkingDirectoryChanges';

export default function RepositoryPanel() {
    const [activeView] = useAtom(activeViewAtom);
    const [currentRepository] = useAtom(selectedRepositoryAtom);
    const [showFileTree, setShowFileTree] = useState(false);
    const isMobile = useIsMobile();

    // Debug logging
    console.log('RepositoryPanel - currentRepository:', currentRepository);
    console.log('RepositoryPanel - activeView:', activeView);

    if (!currentRepository) {
        return (
            <div className="flex-1 flex items-center justify-center text-muted-foreground">
                <div className="text-center">
                    <h2 className="text-lg font-semibold mb-2">No Repository Selected</h2>
                    <p>Select a repository from the sidebar to get started</p>
                </div>
            </div>
        );
    }

    const renderFilesView = () => {
        if (isMobile) {
            return (
                <div className="flex h-full flex-col">
                    {/* Mobile file tree toggle button */}
                    <div className="border-b p-3 bg-muted/30">
                        <Button
                            variant="outline"
                            onClick={() => setShowFileTree(true)}
                            className="w-full justify-start touch-target h-11"
                        >
                            <FolderTree className="h-4 w-4 mr-2" />
                            Browse Files
                        </Button>
                    </div>
                    
                    {/* File viewer takes full space on mobile */}
                    <div className="flex-1">
                        <FileViewer />
                    </div>

                    {/* Mobile file tree drawer */}
                    <Drawer open={showFileTree} onOpenChange={setShowFileTree} direction="bottom">
                        <DrawerContent className="max-h-[85vh] rounded-t-xl">
                            <DrawerHeader className="flex-row items-center justify-between space-y-0 pb-2">
                                <DrawerTitle className="flex items-center gap-2">
                                    <FolderTree className="h-5 w-5" />
                                    Files
                                </DrawerTitle>
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => setShowFileTree(false)}
                                    className="h-8 w-8 p-0"
                                >
                                    <X className="h-4 w-4" />
                                </Button>
                            </DrawerHeader>
                            <div className="flex-1 overflow-hidden">
                                <FileTreeBrowser onFileSelect={() => setShowFileTree(false)} />
                            </div>
                        </DrawerContent>
                    </Drawer>
                </div>
            );
        }

        // Desktop view - side by side
        return (
            <div className="flex h-full">
                <div className="w-80 border-r">
                    <FileTreeBrowser />
                </div>
                <div className="flex-1">
                    <FileViewer />
                </div>
            </div>
        );
    };

    const renderHistoryView = () => <CommitHistory />;

    const renderTreeView = () => <CommitTree />;

    const renderBranchesView = () => <BranchList />;

    const renderStatusView = () => <WorkingDirectoryChanges />;

    const renderSettingsView = () => (
        <div className="p-4">
            <h2 className="text-lg font-semibold mb-4">Settings</h2>
            <p className="text-muted-foreground">
                Application settings will be implemented in Phase 5
            </p>
        </div>
    );

    const renderContent = () => {
        switch (activeView) {
            case 'files':
                return renderFilesView();
            case 'history':
                return renderHistoryView();
            case 'tree':
                return renderTreeView();
            case 'branches':
                return renderBranchesView();
            case 'status':
                return renderStatusView();
            case 'settings':
                return renderSettingsView();
            default:
                return renderFilesView();
        }
    };

    return <div className="flex-1 overflow-hidden">{renderContent()}</div>;
}
