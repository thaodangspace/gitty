import { useAtom } from "jotai";
import { useState } from "react";
import { selectedRepositoryAtom } from "@/store/atoms";
import { useCommitDetails } from "@/store/queries";
import { format } from "date-fns";
import {
  GitCommit,
  User,
  Calendar,
  Hash,
  Plus,
  Minus,
  FileText,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import InlineDiffViewer from "@/components/file/InlineDiffViewer";

export interface CommitDetailsContentProps {
  commitHash: string | null;
}

export default function CommitDetailsContent({ commitHash }: CommitDetailsContentProps) {
  const [currentRepository] = useAtom(selectedRepositoryAtom);
  const [expandedFilePath, setExpandedFilePath] = useState<string | null>(null);

  if (!currentRepository) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-muted-foreground">No repository selected</div>
      </div>
    );
  }

  const {
    data: commitDetails,
    isLoading,
    error,
  } = useCommitDetails(currentRepository.id, commitHash || undefined);

  const getChangeTypeColor = (changeType: string) => {
    switch (changeType) {
      case "added":
        return "bg-green-100 text-green-800 border-green-200";
      case "deleted":
        return "bg-red-100 text-red-800 border-red-200";
      case "modified":
        return "bg-blue-100 text-blue-800 border-blue-200";
      default:
        return "bg-gray-100 text-gray-800 border-gray-200";
    }
  };

  const getChangeTypeIcon = (changeType: string) => {
    switch (changeType) {
      case "added":
        return <Plus className="h-3 w-3" />;
      case "deleted":
        return <Minus className="h-3 w-3" />;
      case "modified":
        return <FileText className="h-3 w-3" />;
      default:
        return <FileText className="h-3 w-3" />;
    }
  };

  const handleFileClick = (filePath: string) => {
    setExpandedFilePath((current) =>
      current === filePath ? null : filePath
    );
  };

  if (isLoading) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-muted-foreground">Loading commit details...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-red-600">
          Error loading commit details: {error.message}
        </div>
      </div>
    );
  }

  if (!commitDetails) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-muted-foreground">Commit not found</div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto">
      {/* Commit Header */}
      <div className="p-6 border-b bg-muted/30">
        <div className="flex items-start gap-4">
          <div className="flex-shrink-0 mt-1">
            <div className="w-10 h-10 bg-primary/10 rounded-full flex items-center justify-center">
              <GitCommit className="h-5 w-5 text-primary" />
            </div>
          </div>

          <div className="flex-1 min-w-0">
            <h3 className="text-lg font-semibold mb-3 leading-relaxed">
              {commitDetails.message}
            </h3>

            <div className="flex items-center gap-6 text-sm text-muted-foreground mb-4">
              <div className="flex items-center gap-2">
                <User className="h-4 w-4" />
                <span className="font-medium">
                  {commitDetails.author.name}
                </span>
                <span className="text-muted-foreground">
                  ({commitDetails.author.email})
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4" />
                <span>
                  {format(
                    new Date(commitDetails.date),
                    "MMM d, yyyy HH:mm:ss",
                  )}
                </span>
              </div>
            </div>

            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2 text-sm bg-muted px-3 py-1 rounded font-mono">
                <Hash className="h-3 w-3" />
                <span>{commitDetails.hash.substring(0, 12)}</span>
              </div>
              {commitDetails.parent_hash && (
                <div className="text-sm text-muted-foreground">
                  Parent:{" "}
                  <span className="font-mono">
                    {commitDetails.parent_hash.substring(0, 12)}
                  </span>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Stats Summary */}
      <div className="p-6 border-b bg-background">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <Badge
              variant="outline"
              className="bg-green-50 text-green-700 border-green-200"
            >
              <Plus className="h-3 w-3 mr-1" />+
              {commitDetails.stats.additions}
            </Badge>
            <Badge
              variant="outline"
              className="bg-red-50 text-red-700 border-red-200"
            >
              <Minus className="h-3 w-3 mr-1" />-
              {commitDetails.stats.deletions}
            </Badge>
          </div>
          <div className="text-sm text-muted-foreground">
            {commitDetails.stats.files_changed} file
            {commitDetails.stats.files_changed !== 1 ? "s" : ""} changed
          </div>
        </div>
      </div>

      {/* File Changes */}
      <div className="p-6">
        <h4 className="text-md font-semibold mb-4">
          Files Changed ({commitDetails.changes.length})
        </h4>

        <div className="space-y-2">
          {commitDetails.changes.map((change) => {
            const isExpanded = expandedFilePath === change.path;
            return (
              <div
                key={change.path}
                className={`border rounded-lg overflow-hidden ${
                  isExpanded ? "ring-1 ring-primary/20" : ""
                }`}
              >
                <button
                  type="button"
                  aria-expanded={isExpanded}
                  onClick={() => handleFileClick(change.path)}
                  className={`w-full flex items-center justify-between p-4 text-left transition-colors focus:outline-none focus:ring-2 focus:ring-primary/20 ${
                    isExpanded
                      ? "bg-muted/50 border-b"
                      : "bg-muted/30 hover:bg-muted/50"
                  }`}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <Badge
                      variant="outline"
                      className={`${getChangeTypeColor(change.change_type)} font-mono text-xs flex-shrink-0`}
                    >
                      {getChangeTypeIcon(change.change_type)}
                      {change.change_type}
                    </Badge>
                    <span className="font-mono text-sm truncate" title={change.path}>
                      {change.path}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 flex-shrink-0 ml-2">
                    {(change.additions > 0 || change.deletions > 0) && (
                      <div className="flex items-center gap-2 text-xs">
                        {change.additions > 0 && (
                          <span className="text-green-600">
                            +{change.additions}
                          </span>
                        )}
                        {change.deletions > 0 && (
                          <span className="text-red-600">
                            -{change.deletions}
                          </span>
                        )}
                      </div>
                    )}
                    {isExpanded ? (
                      <ChevronUp className="h-4 w-4 text-muted-foreground" />
                    ) : (
                      <ChevronDown className="h-4 w-4 text-muted-foreground" />
                    )}
                  </div>
                </button>

                {isExpanded && currentRepository && (
                  <div className="border-t">
                    <InlineDiffViewer
                      repositoryId={currentRepository.id}
                      filePath={change.path}
                      commitHash={commitDetails.hash}
                    />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
