import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import CommitDetailsContent from '@/components/repository/CommitDetailsContent';

export default function CommitDetailsPage() {
  const { repoId, commitHash } = useParams<{ repoId: string; commitHash: string }>();
  const navigate = useNavigate();

  const handleBack = () => {
    navigate(`/repo/${repoId}`);
  };

  return (
    <div className="h-screen flex flex-col bg-background">
      {/* Header with back button */}
      <div className="flex items-center gap-4 p-4 border-b">
        <Button
          variant="ghost"
          size="icon"
          onClick={handleBack}
          aria-label="Back to commit history"
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-lg font-semibold">Commit Details</h1>
      </div>

      {/* Commit content */}
      <CommitDetailsContent commitHash={commitHash || null} />
    </div>
  );
}
