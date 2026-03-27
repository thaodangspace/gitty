import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { GitCommit } from "lucide-react";
import CommitDetailsContent, { CommitDetailsContentProps } from "./CommitDetailsContent";

interface CommitDetailsDialogProps extends CommitDetailsContentProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function CommitDetailsDialog({
  commitHash,
  isOpen,
  onClose,
}: CommitDetailsDialogProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <GitCommit className="h-5 w-5" />
            Commit Details
          </DialogTitle>
        </DialogHeader>

        <CommitDetailsContent commitHash={commitHash} />
      </DialogContent>
    </Dialog>
  );
}
