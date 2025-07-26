import { useAtom } from 'jotai';
import { selectedRepositoryFromListAtom } from '@/store/atoms';

export default function StatusBar() {
    const [currentRepository] = useAtom(selectedRepositoryFromListAtom);

    return (
        <footer className="h-6 border-t bg-muted/50 flex items-center px-4 text-xs text-muted-foreground">
            <div className="flex items-center gap-4">
                {currentRepository && (
                    <>
                        <span>Branch: {currentRepository.current_branch}</span>
                        <span>Path: {currentRepository.path}</span>
                    </>
                )}
                <div className="ml-auto">GitWeb v1.0.0</div>
            </div>
        </footer>
    );
}
