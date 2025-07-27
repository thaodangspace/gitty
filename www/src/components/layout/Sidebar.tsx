import { useAtom } from 'jotai';
import { activeViewAtom, selectedRepositoryAtom } from '@/store/atoms';
import { Button } from '@/components/ui/button';
import { Files, History, GitBranch, GitCommit } from 'lucide-react';
import RepositoryList from '@/components/repository/RepositoryList';
import type { ActiveView } from '@/store/atoms/ui-atoms';

const navigationItems: Array<{ id: ActiveView; label: string; icon: any }> = [
    { id: 'files', label: 'Files', icon: Files },
    { id: 'status', label: 'Status', icon: GitCommit },
    { id: 'history', label: 'History', icon: History },
    { id: 'branches', label: 'Branches', icon: GitBranch },
];

export default function Sidebar() {
    const [activeView, setActiveView] = useAtom(activeViewAtom);
    const [selectedRepository] = useAtom(selectedRepositoryAtom);

    return (
        <div className="h-full flex flex-col">
            <div className="p-4 border-b">
                <RepositoryList />
            </div>

            <nav className="flex-1 p-2">
                <div className="space-y-1">
                    {navigationItems.map((item) => {
                        const Icon = item.icon;
                        const isDisabled = !selectedRepository && item.id !== 'settings';

                        return (
                            <Button
                                key={item.id}
                                variant={activeView === item.id ? 'secondary' : 'ghost'}
                                className="w-full justify-start"
                                onClick={() => setActiveView(item.id)}
                                disabled={isDisabled}
                            >
                                <Icon className="h-4 w-4 mr-2" />
                                {item.label}
                            </Button>
                        );
                    })}
                </div>
            </nav>
        </div>
    );
}
