import { useAtom } from 'jotai';
import { showChooseRepositoryDialogAtom } from '@/store/atoms';
import { Button } from '@/components/ui/button';
import { FolderOpen } from 'lucide-react';

export default function LandingPage() {
    const [, setShowChooseDialog] = useAtom(showChooseRepositoryDialogAtom);

    return (
        <div className="flex-1 flex items-center justify-center bg-background">
            <div className="text-center">
                <h1 className="text-2xl font-semibold mb-4">Gitty</h1>
                <Button size="lg" onClick={() => setShowChooseDialog(true)} className="gap-2">
                    <FolderOpen className="h-5 w-5" />
                    Select Repository
                </Button>
            </div>
        </div>
    );
}
