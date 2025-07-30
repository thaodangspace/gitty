import { useAtom } from 'jotai';
import { sidebarOpenAtom, selectedRepositoryAtom, activeViewAtom } from '@/store/atoms';
import { Button } from '@/components/ui/button';
import { Menu, GitBranch, FolderOpen, History, Settings, GitMerge, RefreshCw, GitCompare, MoreHorizontal } from 'lucide-react';
import { useState } from 'react';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';

export default function Header() {
    const [sidebarOpen, setSidebarOpen] = useAtom(sidebarOpenAtom);
    const [currentRepository] = useAtom(selectedRepositoryAtom);
    const [activeView, setActiveView] = useAtom(activeViewAtom);
    const [showMobileMenu, setShowMobileMenu] = useState(false);

    const navigationItems = [
        { id: 'files', label: 'Files', icon: FolderOpen },
        { id: 'history', label: 'History', icon: History },
        { id: 'branches', label: 'Branches', icon: GitMerge },
        { id: 'status', label: 'Changes', icon: GitCompare },
    ];

    const handleViewChange = (view: string) => {
        setActiveView(view);
        setShowMobileMenu(false);
    };

    return (
        <header className="h-14 md:h-12 border-b bg-background flex items-center px-3 md:px-4 gap-2 md:gap-4">
            {/* Mobile menu button */}
            <Button 
                variant="ghost" 
                size="sm" 
                onClick={() => setSidebarOpen(!sidebarOpen)}
                className="touch-target p-2 md:p-1"
            >
                <Menu className="h-5 w-5 md:h-4 md:w-4" />
            </Button>

            {/* Logo and app name */}
            <div className="flex items-center gap-2 min-w-0 flex-1">
                <GitBranch className="h-5 w-5 md:h-4 md:w-4 flex-shrink-0" />
                <span className="font-semibold text-sm md:text-base truncate">GitWeb</span>
            </div>

            {/* Repository info - hidden on mobile if no space */}
            {currentRepository && (
                <div className="hidden md:flex items-center gap-2 text-sm text-muted-foreground">
                    <span>/</span>
                    <span className="font-medium truncate max-w-32">{currentRepository.name}</span>
                    <span className="text-xs bg-muted px-2 py-1 rounded flex-shrink-0">
                        {currentRepository.current_branch}
                    </span>
                </div>
            )}

            {/* Desktop navigation */}
            <div className="hidden md:flex items-center gap-1 ml-4">
                {currentRepository && navigationItems.map((item) => {
                    const Icon = item.icon;
                    return (
                        <Button 
                            key={item.id}
                            variant={activeView === item.id ? 'secondary' : 'ghost'} 
                            size="sm"
                            onClick={() => setActiveView(item.id)}
                            className="touch-target"
                        >
                            <Icon className="h-4 w-4 mr-1" />
                            {item.label}
                        </Button>
                    );
                })}
            </div>

            {/* Mobile navigation dropdown */}
            {currentRepository && (
                <div className="md:hidden">
                    <DropdownMenu open={showMobileMenu} onOpenChange={setShowMobileMenu}>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm" className="touch-target p-2">
                                <MoreHorizontal className="h-5 w-5" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-48">
                            {navigationItems.map((item) => {
                                const Icon = item.icon;
                                return (
                                    <DropdownMenuItem 
                                        key={item.id}
                                        onClick={() => handleViewChange(item.id)}
                                        className="touch-target"
                                    >
                                        <Icon className="h-4 w-4 mr-2" />
                                        {item.label}
                                    </DropdownMenuItem>
                                );
                            })}
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            )}

            {/* Right side actions */}
            <div className="flex items-center gap-1 md:gap-2 ml-auto">
                {currentRepository && (
                    <Button 
                        variant="ghost" 
                        size="sm" 
                        title="Refresh"
                        className="touch-target p-2 md:p-1"
                    >
                        <RefreshCw className="h-5 w-5 md:h-4 md:w-4" />
                    </Button>
                )}
                <Button 
                    variant={activeView === 'settings' ? 'secondary' : 'ghost'} 
                    size="sm"
                    onClick={() => setActiveView('settings')}
                    title="Settings"
                    className="touch-target p-2 md:p-1"
                >
                    <Settings className="h-5 w-5 md:h-4 md:w-4" />
                </Button>
            </div>
        </header>
    );
}
