import { useAtom } from 'jotai';
import { activeViewAtom, selectedRepositoryFromListAtom } from '@/store/atoms';

export default function MainContent() {
    const [activeView] = useAtom(activeViewAtom);
    const [currentRepository] = useAtom(selectedRepositoryFromListAtom);

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

    const renderContent = () => {
        switch (activeView) {
            case 'files':
                return (
                    <div className="p-4">
                        <h2 className="text-lg font-semibold mb-4">Files</h2>
                        <p className="text-muted-foreground">
                            File browser will be implemented here
                        </p>
                    </div>
                );
            case 'history':
                return (
                    <div className="p-4">
                        <h2 className="text-lg font-semibold mb-4">Commit History</h2>
                        <p className="text-muted-foreground">
                            Commit history will be implemented here
                        </p>
                    </div>
                );
            case 'branches':
                return (
                    <div className="p-4">
                        <h2 className="text-lg font-semibold mb-4">Branches</h2>
                        <p className="text-muted-foreground">
                            Branch management will be implemented here
                        </p>
                    </div>
                );
            case 'settings':
                return (
                    <div className="p-4">
                        <h2 className="text-lg font-semibold mb-4">Settings</h2>
                        <p className="text-muted-foreground">
                            Application settings will be implemented here
                        </p>
                    </div>
                );
            default:
                return null;
        }
    };

    return <div className="flex-1 overflow-auto">{renderContent()}</div>;
}
