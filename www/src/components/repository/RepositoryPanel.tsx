import { useAtom } from 'jotai';
import { activeViewAtom, selectedRepositoryAtom } from '@/store/atoms';
import FileTreeBrowser from '../file/FileTreeBrowser';
import FileViewer from '../file/FileViewer';
import CommitHistory from './CommitHistory';
import BranchList from './BranchList';
import WorkingDirectoryChanges from './WorkingDirectoryChanges';

export default function RepositoryPanel() {
    const [activeView] = useAtom(activeViewAtom);
    const [currentRepository] = useAtom(selectedRepositoryAtom);

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

    const renderFilesView = () => (
        <div className="flex h-full">
            <div className="w-80 border-r">
                <FileTreeBrowser />
            </div>
            <div className="flex-1">
                <FileViewer />
            </div>
        </div>
    );

    const renderHistoryView = () => <CommitHistory />;

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
