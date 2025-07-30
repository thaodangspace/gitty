import { useMemo, useState } from 'react';
import { useAtom } from 'jotai';
import { selectedRepositoryAtom } from '@/store/atoms';
import { useCommitHistory } from '@/store/queries';
import { format } from 'date-fns';
import { GitBranch, GitCommit, User, Calendar, Hash, GitMerge } from 'lucide-react';
import { Button } from '@/components/ui/button';
import CommitDetailsDialog from './CommitDetailsDialog';

interface CommitNode {
  hash: string;
  message: string;
  author: {
    name: string;
    email: string;
  };
  date: Date;
  parentHash?: string;
  x: number;
  y: number;
  branch: number;
  isMerge: boolean;
}

interface BranchLine {
  branch: number;
  color: string;
  commits: string[];
}

const BRANCH_COLORS = [
  '#22c55e', // green
  '#3b82f6', // blue  
  '#f59e0b', // yellow
  '#ef4444', // red
  '#8b5cf6', // purple
  '#06b6d4', // cyan
  '#f97316', // orange
  '#84cc16', // lime
];

export default function CommitTree() {
  const [currentRepository] = useAtom(selectedRepositoryAtom);
  const { data: commits, isLoading, error } = useCommitHistory(currentRepository?.id);
  const [selectedCommitHash, setSelectedCommitHash] = useState<string | null>(null);

  const { nodes, branches } = useMemo(() => {
    if (!commits || commits.length === 0) {
      return { nodes: [], branches: [] };
    }

    // Create a map of commits for easy lookup
    const commitMap = new Map(commits.map(c => [c.hash, c]));
    
    // Build a graph of parent-child relationships
    const children = new Map<string, string[]>();
    commits.forEach(commit => {
      if (commit.parent_hash) {
        if (!children.has(commit.parent_hash)) {
          children.set(commit.parent_hash, []);
        }
        children.get(commit.parent_hash)!.push(commit.hash);
      }
    });

    // Simple branch assignment algorithm
    // This is a simplified version - a real implementation would need more sophisticated branch tracking
    const branchAssignments = new Map<string, number>();
    const branchLines: BranchLine[] = [];
    let currentBranch = 0;

    // Start with the first commit (most recent)
    commits.forEach((commit, index) => {
      const childrenHashes = children.get(commit.hash) || [];
      const hasMultipleChildren = childrenHashes.length > 1;
      const isSingleParent = !commit.parent_hash || (commitMap.get(commit.parent_hash)?.parent_hash);
      
      // If this commit doesn't have a branch assignment yet
      if (!branchAssignments.has(commit.hash)) {
        // If it's a merge commit or has multiple children, it might be on main branch
        if (hasMultipleChildren || index < 3) {
          branchAssignments.set(commit.hash, 0);
        } else {
          // Try to inherit from parent
          if (commit.parent_hash && branchAssignments.has(commit.parent_hash)) {
            branchAssignments.set(commit.hash, branchAssignments.get(commit.parent_hash)!);
          } else {
            branchAssignments.set(commit.hash, currentBranch++);
          }
        }
      }
    });

    // Create nodes with positions
    const nodeHeight = 60;
    const nodeWidth = 40;
    const nodes: CommitNode[] = commits.map((commit, index) => {
      const branch = branchAssignments.get(commit.hash) || 0;
      const childrenCount = children.get(commit.hash)?.length || 0;
      
      return {
        hash: commit.hash,
        message: commit.message,
        author: commit.author,
        date: new Date(commit.date),
        parentHash: commit.parent_hash,
        x: branch * nodeWidth + 20,
        y: index * nodeHeight + 30,
        branch,
        isMerge: childrenCount > 1,
      };
    });

    // Create branch lines
    const activeBranches = new Set(nodes.map(n => n.branch));
    activeBranches.forEach(branchNum => {
      const branchCommits = nodes.filter(n => n.branch === branchNum).map(n => n.hash);
      branchLines.push({
        branch: branchNum,
        color: BRANCH_COLORS[branchNum % BRANCH_COLORS.length],
        commits: branchCommits,
      });
    });

    return { nodes, branches: branchLines };
  }, [commits]);

  if (isLoading) {
    return (
      <div className="p-4 flex items-center justify-center">
        <div className="text-muted-foreground">Loading commit tree...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 flex items-center justify-center">
        <div className="text-red-600">Error loading commits: {error.message}</div>
      </div>
    );
  }

  if (!commits || commits.length === 0) {
    return (
      <div className="p-4 flex items-center justify-center">
        <div className="text-center text-muted-foreground">
          <GitBranch className="h-12 w-12 mx-auto mb-2 opacity-50" />
          <h3 className="text-lg font-medium mb-1">No Commit Tree</h3>
          <p>This repository doesn't have any commits yet</p>
        </div>
      </div>
    );
  }

  const svgWidth = Math.max(400, (branches.length * 40) + 60);
  const svgHeight = Math.max(300, nodes.length * 60 + 60);

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b bg-muted/50">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <GitBranch className="h-5 w-5" />
            Commit Tree
          </h2>
          <div className="text-sm text-muted-foreground">
            {commits.length} commits • {branches.length} branches
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-auto bg-background">
        <div className="relative min-w-full">
          <svg
            width={svgWidth}
            height={svgHeight}
            className="absolute top-0 left-0"
            style={{ zIndex: 1 }}
          >
            {/* Draw branch lines */}
            {branches.map((branch) => (
              <g key={branch.branch}>
                {nodes
                  .filter(n => n.branch === branch.branch)
                  .map((node, index, branchNodes) => {
                    if (index === branchNodes.length - 1) return null;
                    const nextNode = branchNodes[index + 1];
                    return (
                      <line
                        key={`${node.hash}-${nextNode.hash}`}
                        x1={node.x}
                        y1={node.y}
                        x2={nextNode.x}
                        y2={nextNode.y}
                        stroke={branch.color}
                        strokeWidth="2"
                        opacity="0.7"
                      />
                    );
                  })}
              </g>
            ))}

            {/* Draw merge lines */}
            {nodes.map((node) => {
              if (!node.parentHash) return null;
              const parent = nodes.find(n => n.hash === node.parentHash);
              if (!parent || parent.branch === node.branch) return null;
              
              return (
                <line
                  key={`merge-${node.hash}-${parent.hash}`}
                  x1={node.x}
                  y1={node.y}
                  x2={parent.x}
                  y2={parent.y}
                  stroke="#64748b"
                  strokeWidth="1"
                  strokeDasharray="4,4"
                  opacity="0.5"
                />
              );
            })}

            {/* Draw commit nodes */}
            {nodes.map((node) => (
              <g key={node.hash}>
                <circle
                  cx={node.x}
                  cy={node.y}
                  r="6"
                  fill={BRANCH_COLORS[node.branch % BRANCH_COLORS.length]}
                  stroke="white"
                  strokeWidth="2"
                  className="cursor-pointer hover:r-8 transition-all"
                  onClick={() => setSelectedCommitHash(node.hash)}
                />
                {node.isMerge && (
                  <GitMerge
                    x={node.x - 6}
                    y={node.y - 16}
                    className="h-3 w-3 text-muted-foreground"
                  />
                )}
              </g>
            ))}
          </svg>

          {/* Commit details overlay */}
          <div className="relative" style={{ zIndex: 2 }}>
            {nodes.map((node) => (
              <div
                key={node.hash}
                className="absolute border rounded-lg bg-background/95 backdrop-blur-sm shadow-sm hover:shadow-md transition-shadow cursor-pointer"
                style={{
                  left: node.x + 20,
                  top: node.y - 20,
                  width: '300px',
                }}
                onClick={() => setSelectedCommitHash(node.hash)}
              >
                <div className="p-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <h3 className="font-medium text-sm leading-5 mb-1 truncate">
                        {node.message}
                      </h3>
                      
                      <div className="flex items-center gap-3 text-xs text-muted-foreground">
                        <div className="flex items-center gap-1">
                          <User className="h-3 w-3" />
                          <span className="truncate">{node.author.name}</span>
                        </div>
                        <div className="flex items-center gap-1">
                          <Calendar className="h-3 w-3" />
                          <span>{format(node.date, 'MMM d, HH:mm')}</span>
                        </div>
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-1 flex-shrink-0">
                      <div className="flex items-center gap-1 text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded font-mono">
                        <Hash className="h-3 w-3" />
                        <span>{node.hash.substring(0, 7)}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
      
      <CommitDetailsDialog
        commitHash={selectedCommitHash}
        isOpen={!!selectedCommitHash}
        onClose={() => setSelectedCommitHash(null)}
      />
    </div>
  );
}