import { useAtom } from 'jotai';
import { sidebarOpenAtom, selectedRepositoryFromListAtom, activeViewAtom } from '@/store/atoms';
import { Button } from '@/components/ui/button';
import { Menu, GitBranch, FolderOpen, History, Settings, GitMerge, RefreshCw, GitCompare } from 'lucide-react';

export default function Header() {
    const [sidebarOpen, setSidebarOpen] = useAtom(sidebarOpenAtom);
    const [currentRepository] = useAtom(selectedRepositoryFromListAtom);
    const [activeView, setActiveView] = useAtom(activeViewAtom);

    return (
        <header className="h-12 border-b bg-background flex items-center px-4 gap-4">
            <Button variant="ghost" size="sm" onClick={() => setSidebarOpen(!sidebarOpen)}>
                <Menu className="h-4 w-4" />
            </Button>

            <div className="flex items-center gap-2">
                <GitBranch className="h-4 w-4" />
                <span className="font-semibold">GitWeb</span>
            </div>

            {currentRepository && (
                <>
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <span>/</span>
                        <span className="font-medium">{currentRepository.name}</span>
                        <span className="text-xs bg-muted px-2 py-1 rounded">
                            {currentRepository.current_branch}
                        </span>
                    </div>

                    <div className="flex items-center gap-1 ml-4">
                        <Button 
                            variant={activeView === 'files' ? 'secondary' : 'ghost'} 
                            size="sm"
                            onClick={() => setActiveView('files')}
                        >
                            <FolderOpen className="h-4 w-4 mr-1" />
                            Files
                        </Button>
                        <Button 
                            variant={activeView === 'history' ? 'secondary' : 'ghost'} 
                            size="sm"
                            onClick={() => setActiveView('history')}
                        >
                            <History className="h-4 w-4 mr-1" />
                            History
                        </Button>
                        <Button 
                            variant={activeView === 'branches' ? 'secondary' : 'ghost'} 
                            size="sm"
                            onClick={() => setActiveView('branches')}
                        >
                            <GitMerge className="h-4 w-4 mr-1" />
                            Branches
                        </Button>
                        <Button 
                            variant={activeView === 'status' ? 'secondary' : 'ghost'} 
                            size="sm"
                            onClick={() => setActiveView('status')}
                        >
                            <GitCompare className="h-4 w-4 mr-1" />
                            Changes
                        </Button>
                    </div>
                </>
            )}

            <div className="ml-auto flex items-center gap-2">
                {currentRepository && (
                    <Button variant="ghost" size="sm" title="Refresh">
                        <RefreshCw className="h-4 w-4" />
                    </Button>
                )}
                <Button 
                    variant={activeView === 'settings' ? 'secondary' : 'ghost'} 
                    size="sm"
                    onClick={() => setActiveView('settings')}
                    title="Settings"
                >
                    <Settings className="h-4 w-4" />
                </Button>
            </div>
        </header>
    );
}
