import { useAtom } from "jotai";
import {
  selectedRepositoryAtom,
  statusMessageAtom,
  progressAtom,
} from "@/store/atoms";
import { useRepositoryStatus } from "@/store/queries";
import { GitBranch, Circle, AlertCircle, CheckCircle2 } from "lucide-react";

export default function StatusBar() {
  const [currentRepository] = useAtom(selectedRepositoryAtom);
  const [statusMessage] = useAtom(statusMessageAtom);
  const [progress] = useAtom(progressAtom);

  const { data: repoStatus } = useRepositoryStatus(currentRepository?.id);

  const getStatusIcon = () => {
    if (!repoStatus) return null;

    if (repoStatus.conflicts.length > 0) {
      return <AlertCircle className="h-3 w-3 text-red-500" />;
    }

    if (repoStatus.is_clean) {
      return <CheckCircle2 className="h-3 w-3 text-green-500" />;
    }

    return <Circle className="h-3 w-3 text-yellow-500 fill-current" />;
  };

  const getChangesText = () => {
    if (!repoStatus) return null;

    const staged = repoStatus?.staged?.length;
    const modified = repoStatus?.modified?.length;
    const untracked = repoStatus?.untracked?.length;

    if (staged === 0 && modified === 0 && untracked === 0) {
      return "No changes";
    }

    const parts = [];
    if (staged > 0) parts.push(`${staged} staged`);
    if (modified > 0) parts.push(`${modified} modified`);
    if (untracked > 0) parts.push(`${untracked} untracked`);

    return parts.join(", ");
  };

  return (
    <footer className="h-8 md:h-6 border-t bg-muted/50 flex items-center px-3 md:px-4 text-xs text-muted-foreground">
      <div className="flex items-center gap-2 md:gap-4 flex-1 min-w-0">
        {currentRepository && (
          <>
            {/* Branch info - responsive */}
            <div className="flex items-center gap-1 flex-shrink-0">
              <GitBranch className="h-3 w-3" />
              <span className="truncate max-w-20 md:max-w-none">
                {currentRepository.current_branch}
              </span>
            </div>

            {repoStatus && (
              <>
                {/* Status info - hidden on very small screens */}
                <div className="hidden sm:flex items-center gap-1">
                  {getStatusIcon()}
                  <span className="truncate">{getChangesText()}</span>
                </div>

                {/* Sync status - always visible */}
                {(repoStatus.ahead > 0 || repoStatus.behind > 0) && (
                  <div className="flex items-center gap-1 flex-shrink-0">
                    {repoStatus.ahead > 0 && (
                      <span className="text-blue-500">↑{repoStatus.ahead}</span>
                    )}
                    {repoStatus.behind > 0 && (
                      <span className="text-orange-500">
                        ↓{repoStatus.behind}
                      </span>
                    )}
                  </div>
                )}
              </>
            )}

            {/* Repository path - hidden on mobile */}
            <span className="hidden lg:block truncate flex-1">
              Path: {currentRepository.path}
            </span>
          </>
        )}

        {/* Right side - responsive */}
        <div className="flex items-center gap-2 ml-auto flex-shrink-0">
          {progress && (
            <div className="hidden sm:flex items-center gap-1">
              <div className="w-12 md:w-16 h-1 bg-muted rounded-full overflow-hidden">
                <div
                  className="h-full bg-primary transition-all duration-300"
                  style={{
                    width: `${(progress.current / progress.total) * 100}%`,
                  }}
                />
              </div>
              <span className="text-xs">
                {progress.current}/{progress.total}
              </span>
            </div>
          )}

          {/* Status message - responsive */}
          <span className="truncate max-w-24 md:max-w-48 lg:max-w-none">
            {statusMessage}
          </span>
          <span className="opacity-60 hidden md:block">Gitty v1.0.0</span>
        </div>
      </div>
    </footer>
  );
}
