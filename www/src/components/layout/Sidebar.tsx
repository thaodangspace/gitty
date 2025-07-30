import { useAtom } from 'jotai';
import { selectedRepositoryAtom } from '@/store/atoms';
import { useBranches } from '@/store/queries';
import { ChevronDown, ChevronRight, Star, GitBranch, Folder, Check, Filter, Search, X } from 'lucide-react';
import RepositoryList from '@/components/repository/RepositoryList';
import { useState } from 'react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

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
                <div className="p-3 md:p-4 border-b">
                    <RepositoryList />
                </div>
                <div className="flex-1 flex items-center justify-center text-muted-foreground p-4 text-center">
                    <div>
                        <p className="text-sm">Select a repository to view branches</p>
                        <p className="text-xs mt-1">Tap the menu button to browse repositories</p>
                    </div>
                </div>
            </div>
        );
    }

    if (isLoading) {
        return (
            <div className="h-full flex flex-col">
                <div className="p-3 md:p-4 border-b">
                    <RepositoryList />
                </div>
                <div className="flex-1 flex items-center justify-center p-4">
                    <div className="text-muted-foreground text-center">
                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mx-auto mb-2"></div>
                        <p className="text-sm">Loading branches...</p>
                    </div>
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
        return acc;
    }, {} as Record<string, BranchItem[]>);

    return (
        <div className="h-full flex flex-col">
            {/* Header with close button on mobile */}
            <div className="p-3 md:p-4 border-b flex items-center justify-between">
                <div className="flex-1">
                    <RepositoryList />
                </div>
                <Button 
                    variant="ghost" 
                    size="sm" 
                    className="md:hidden touch-target p-2"
                    onClick={() => {
                        // Close sidebar on mobile
                        const event = new CustomEvent('closeSidebar');
                        window.dispatchEvent(event);
                    }}
                >
                    <X className="h-4 w-4" />
                </Button>
            </div>

            <div className="flex-1 flex flex-col min-h-0">
                {/* Filter Bar */}
                <div className="p-3 border-b">
                    <div className="relative">
                        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                        <Input
                            placeholder="Filter branches..."
                            value={filterQuery}
                            onChange={(e) => setFilterQuery(e.target.value)}
                            className="pl-9 h-10 md:h-9"
                        />
                    </div>
                </div>

                <div className="flex-1 overflow-y-auto overscroll-contain">
                    {/* Starred Section */}
                    <div className="px-3 py-2">
                        <button
                            onClick={() => toggleSection('starred')}
                            className="flex items-center gap-2 w-full text-left text-sm font-medium hover:bg-muted/50 rounded px-2 py-2 touch-target"
                        >
                            {expandedSections.starred ? (
                                <ChevronDown className="h-4 w-4 flex-shrink-0" />
                            ) : (
                                <ChevronRight className="h-4 w-4 flex-shrink-0" />
                            )}
                            <span>Starred</span>
                        </button>
                        {expandedSections.starred && currentBranch && (
                            <div className="ml-6 mt-1">
                                <div className="flex items-center gap-2 py-2 px-2 rounded hover:bg-muted/50 touch-target">
                                    <Check className="h-4 w-4 text-green-600 flex-shrink-0" />
                                    <span className="text-sm truncate">{currentBranch.name}</span>
                                    <span className="text-xs text-muted-foreground ml-auto flex-shrink-0">853↓</span>
                                    <Star className="h-4 w-4 text-blue-500 flex-shrink-0" />
                                    <Filter className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                                </div>
                            </div>
                        )}
                    </div>

                    {/* Branches Section */}
                    <div className="px-3 py-2">
                        <button
                            onClick={() => toggleSection('branches')}
                            className="flex items-center gap-2 w-full text-left text-sm font-medium hover:bg-muted/50 rounded px-2 py-2 touch-target"
                        >
                            {expandedSections.branches ? (
                                <ChevronDown className="h-4 w-4 flex-shrink-0" />
                            ) : (
                                <ChevronRight className="h-4 w-4 flex-shrink-0" />
                            )}
                            <span>Branches</span>
                        </button>
                        {expandedSections.branches && (
                            <div className="ml-6 mt-1 space-y-1">
                                {currentBranch && (
                                    <div className="flex items-center gap-2 py-2 px-2 rounded hover:bg-muted/50 touch-target">
                                        <Check className="h-4 w-4 text-green-600 flex-shrink-0" />
                                        <span className="text-sm truncate">{currentBranch.name}</span>
                                        <span className="text-xs text-muted-foreground ml-auto flex-shrink-0">853↓</span>
                                    </div>
                                )}
                                {localBranches.map((branch) => (
                                    <div key={branch.name} className="flex items-center gap-2 py-2 px-2 rounded hover:bg-muted/50 touch-target">
                                        <GitBranch className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                                        <span className="text-sm truncate">{branch.name}</span>
                                        {branch.last_commit && (
                                            <span className="text-xs text-muted-foreground ml-auto flex-shrink-0">
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
                            className="flex items-center gap-2 w-full text-left text-sm font-medium hover:bg-muted/50 rounded px-2 py-2 touch-target"
                        >
                            {expandedSections.remotes ? (
                                <ChevronDown className="h-4 w-4 flex-shrink-0" />
                            ) : (
                                <ChevronRight className="h-4 w-4 flex-shrink-0" />
                            )}
                            <span>Remotes</span>
                        </button>
                        {expandedSections.remotes && (
                            <div className="ml-6 mt-1 space-y-1">
                                {Object.entries(remoteGroups).map(([remoteName, branches]) => (
                                    <div key={remoteName}>
                                        <div className="flex items-center gap-2 py-2 px-2 rounded hover:bg-muted/50 touch-target">
                                            <Folder className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                                            <span className="text-sm truncate">{remoteName}</span>
                                            <ChevronRight className="h-4 w-4 text-muted-foreground ml-auto flex-shrink-0" />
                                        </div>
                                        <div className="ml-4 space-y-1">
                                            {branches.map((branch) => (
                                                <div key={branch.name} className="flex items-center gap-2 py-2 px-2 rounded hover:bg-muted/50 touch-target">
                                                    <GitBranch className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                                                    <span className="text-sm truncate">{branch.name.split('/')[1]}</span>
                                                    {branch.last_commit && (
                                                        <span className="text-xs text-muted-foreground ml-auto flex-shrink-0">
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
