interface FolderSelectionDialogProps {
  onSelectPath: (path: string) => void;
  title?: string;
  description?: string;
}

export default function FolderSelectionDialog({ 
  onSelectPath, 
  title = "Select Folder",
  description = "Choose a folder for your repository"
}: FolderSelectionDialogProps) {
  // Temporarily disabled due to missing UI components
  // This component needs Dialog, ScrollArea and other shadcn/ui components to be installed
  console.log('FolderSelectionDialog called with:', { onSelectPath, title, description });
  return null;
}