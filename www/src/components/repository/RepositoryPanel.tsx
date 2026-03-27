import { useAtom } from 'jotai';
import {
    activeViewAtom,
    selectedRepositoryAtom,
    showChooseRepositoryDialogAtom,
} from '@/store/atoms';
import { Button } from '@/components/ui/button';
import { FolderOpen, Settings } from 'lucide-react';
import CommitHistory from './CommitHistory';
import BranchList from './BranchList';
import WorkingDirectoryChanges from './WorkingDirectoryChanges';
import { ThemeToggle } from '@/components/ui/theme-toggle';

export default function RepositoryPanel() {
    const [activeView] = useAtom(activeViewAtom);
    const [currentRepository] = useAtom(selectedRepositoryAtom);
    const [, setShowChooseDialog] = useAtom(showChooseRepositoryDialogAtom);

    console.log('RepositoryPanel - currentRepository:', currentRepository);
    console.log('RepositoryPanel - activeView:', activeView);

    if (!currentRepository) {
        return (
            <div className="flex-1 flex items-center justify-center">
                <Button size="lg" onClick={() => setShowChooseDialog(true)} className="gap-2">
                    <FolderOpen className="h-5 w-5" />
                    Select Repository
                </Button>
            </div>
        );
    }


    const renderHistoryView = () => <CommitHistory />;

    const renderBranchesView = () => <BranchList />;

    const renderStatusView = () => <WorkingDirectoryChanges />;

    const renderSettingsView = () => (
        <div className="p-6 max-w-2xl mx-auto">
            <div className="flex items-center gap-3 mb-6">
                <Settings className="h-6 w-6" />
                <h2 className="text-xl font-semibold">Settings</h2>
            </div>
            
            <div className="space-y-6">
                <div className="bg-card rounded-lg border p-4">
                    <h3 className="font-medium mb-2">Appearance</h3>
                    <div className="flex items-center justify-between">
                        <span className="text-sm text-muted-foreground">Theme</span>
                        <ThemeToggle />
                    </div>
                </div>
                
                <div className="text-sm text-muted-foreground">
                    <p>Application settings will be implemented in Phase 5</p>
                </div>
            </div>
        </div>
    );

    const renderContent = () => {
        switch (activeView) {
            case 'history':
                return renderHistoryView();
            case 'branches':
                return renderBranchesView();
            case 'status':
            case 'files': // No need for files view for now
            default:
                return renderStatusView();
            case 'settings':
                return renderSettingsView();
        }
    };

    return <div className="flex-1 overflow-hidden">{renderContent()}</div>;
}
