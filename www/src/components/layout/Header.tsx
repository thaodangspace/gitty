import { useAtom } from 'jotai';
import { sidebarOpenAtom, selectedRepositoryFromListAtom } from '@/store/atoms';
import { Button } from '@/components/ui/button';
import { Menu, GitBranch } from 'lucide-react';

export default function Header() {
    const [sidebarOpen, setSidebarOpen] = useAtom(sidebarOpenAtom);
    const [currentRepository] = useAtom(selectedRepositoryFromListAtom);

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
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <span>/</span>
                    <span className="font-medium">{currentRepository.name}</span>
                    <span className="text-xs bg-muted px-2 py-1 rounded">
                        {currentRepository.current_branch}
                    </span>
                </div>
            )}

            <div className="ml-auto flex items-center gap-2">
                {/* Future: User menu, settings, etc. */}
            </div>
        </header>
    );
}
