import { useAtom } from 'jotai';
import { selectedRepositoryAtom } from '@/store/atoms';
import { useBranches } from '@/store/queries';
import { ChevronDown, ChevronRight, Star, GitBranch, Folder, Check, Filter, Search } from 'lucide-react';
import RepositoryList from '@/components/repository/RepositoryList';
import { useState } from 'react';
import { Input } from '@/components/ui/input';

interface BranchItem {
    name: string;
    is_current: boolean;
    is_remote: boolean;
    last_commit?: {
        hash: string;
        message: string;
        date: string;
        author: { name: string };
    };
}

export default function Sidebar() {
    const [selectedRepository] = useAtom(selectedRepositoryAtom);
    const { data: branches, isLoading } = useBranches(selectedRepository?.id);
    const [filterQuery, setFilterQuery] = useState('');
    const [expandedSections, setExpandedSections] = useState({
        starred: true,
        branches: true,
        remotes: true
    });

    const toggleSection = (section: keyof typeof expandedSections) => {
        setExpandedSections(prev => ({
            ...prev,
            [section]: !prev[section]
        }));
    };

    if (!selectedRepository) {
        return (
            <div className="h-full flex flex-col">
                <div className="p-4 border-b">
                    <RepositoryList />
                </div>
                <div className="flex-1 flex items-center justify-center text-muted-foreground">
                    Select a repository to view branches
                </div>
            </div>
        );
    }

    if (isLoading) {
        return (
            <div className="h-full flex flex-col">
                <div className="p-4 border-b">
                    <RepositoryList />
                </div>
                <div className="flex-1 flex items-center justify-center">
                    <div className="text-muted-foreground">Loading branches...</div>
                </div>
            </div>
        );
    }

    const currentBranch = branches?.find(b => b.is_current);
    const localBranches = branches?.filter(b => !b.is_remote && !b.is_current) || [];
    const remoteBranches = branches?.filter(b => b.is_remote) || [];

    // Group remote branches by remote name
    const remoteGroups = remoteBranches.reduce((acc, branch) => {
        const remoteName = branch.name.split('/')[0];
        if (!acc[remoteName]) {
            acc[remoteName] = [];
        }
        acc[remoteName].push(branch);
        return acc    }, {} as Record<string, BranchItem[]>);

    return (
        <div className="h-full flex flex-col">
            <div className="p-4 border-b">
                <RepositoryList />
            </div>

            <div className="flex-1 flex flex-col">
                {/* Filter Bar */}
                <div className="p-3 border-b">
                    <div className="relative">
                        <Search className="absolute left-2 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                        <Input
                            placeholder="Filter"
                            value={filterQuery}
                            onChange={(e) => setFilterQuery(e.target.value)}
                            className="pl-8"
                        />
                    </div>
                </div>

                <div className="flex-1 overflow-auto">
                    {/* Starred Section */}
                    <div className="px-3 py-2">
                        <button
                            onClick={() => toggleSection('starred')}
                            className="flex items-center gap-2 w-full text-left text-sm font-medium hover:bg-muted/50 rounded px-2 py-1"
                        >
                            {expandedSections.starred ? (
                                <ChevronDown className="h-4 w-4" />
                            ) : (
                                <ChevronRight className="h-4 w-4" />
                            )}
                            Starred
                        </button>
                        {expandedSections.starred && currentBranch && (
                            <div className="ml-6 mt-1">
                                <div className="flex items-center gap-2 py-1 px-2 rounded hover:bg-muted/50">
                                    <Check className="h-4 w-4 text-green-600" />
                                    <span className="text-sm">{currentBranch.name}</span>
                                    <span className="text-xs text-muted-foreground ml-auto">853↓</span>
                                    <Star className="h-4 w-4 text-blue-500" />
                                    <Filter className="h-4 w-4 text-muted-foreground" />
                                </div>
                            </div>
                        )}
                    </div>

                    {/* Branches Section */}
                    <div className="px-3 py-2">
                        <button
                            onClick={() => toggleSection('branches')}
                            className="flex items-center gap-2 w-full text-left text-sm font-medium hover:bg-muted/50 rounded px-2 py-1"
                        >
                            {expandedSections.branches ? (
                                <ChevronDown className="h-4 w-4" />
                            ) : (
                                <ChevronRight className="h-4 w-4" />
                            )}
                            Branches
                        </button>
                        {expandedSections.branches && (
                            <div className="ml-6 mt-1 space-y-1">
                                {currentBranch && (
                                    <div className="flex items-center gap-2 py-1 px-2 rounded hover:bg-muted/50">
                                        <Check className="h-4 w-4 text-green-600" />
                                        <span className="text-sm">{currentBranch.name}</span>
                                        <span className="text-xs text-muted-foreground ml-auto">853↓</span>
                                    </div>
                                )}
                                {localBranches.map((branch) => (
                                    <div key={branch.name} className="flex items-center gap-2 py-1 px-2 rounded hover:bg-muted/50">
                                        <GitBranch className="h-4 w-4 text-muted-foreground" />
                                        <span className="text-sm">{branch.name}</span>
                                        {branch.last_commit && (
                                            <span className="text-xs text-muted-foreground ml-auto">
                                                {branch.last_commit.hash.substring(0, 7)}
                                            </span>
                                        )}
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Remotes Section */}
                    <div className="px-3 py-2">
                        <button
                            onClick={() => toggleSection('remotes')}
                            className="flex items-center gap-2 w-full text-left text-sm font-medium hover:bg-muted/50 rounded px-2 py-1"
                        >
                            {expandedSections.remotes ? (
                                <ChevronDown className="h-4 w-4" />
                            ) : (
                                <ChevronRight className="h-4 w-4" />
                            )}
                            Remotes
                        </button>
                        {expandedSections.remotes && (
                            <div className="ml-6 mt-1 space-y-1">
                                {Object.entries(remoteGroups).map(([remoteName, branches]) => (
                                    <div key={remoteName}>
                                        <div className="flex items-center gap-2 py-1 px-2 rounded hover:bg-muted/50">
                                            <Folder className="h-4 w-4 text-muted-foreground" />
                                            <span className="text-sm">{remoteName}</span>
                                            <ChevronRight className="h-4 w-4 text-muted-foreground ml-auto" />
                                        </div>
                                        <div className="ml-4 space-y-1">
                                            {branches.map((branch) => (
                                                <div key={branch.name} className="flex items-center gap-2 py-1 px-2 rounded hover:bg-muted/50">
                                                    <GitBranch className="h-4 w-4 text-muted-foreground" />
                                                    <span className="text-sm">{branch.name.split('/')[1]}</span>
                                                    {branch.last_commit && (
                                                        <span className="text-xs text-muted-foreground ml-auto">
                                                            {branch.last_commit.hash.substring(0, 7)}
                                                        </span>
                                                    )}
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
