import { useQuery, useMutation, useQueryClient, QueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { useRef } from 'react';

export const useRepositories = () => {
    return useQuery({
        queryKey: ['repositories'],
        queryFn: () => apiClient.getRepositories(),
    });
};

export const useRepository = (id: string | undefined) => {
    return useQuery({
        queryKey: ['repository', id],
        queryFn: () => apiClient.getRepository(id!),
        enabled: !!id,
    });
};

const repositoryStatusKey = (id: string | undefined) => ['repository-status', id] as const;

export const triggerRepositoryStatus = async (
    queryClient: QueryClient,
    repositoryId: string | undefined
) => {
    if (!repositoryId) {
        return;
    }

    try {
        const status = await apiClient.getRepositoryStatus(repositoryId, false);
        queryClient.setQueryData(repositoryStatusKey(repositoryId), status);
    } catch (error) {
        console.error('Failed to trigger repository status refresh', error);
    }
};

export const useRepositoryStatus = (id: string | undefined) => {
    const initialFetchCompleted = useRef(false);

    return useQuery({
        queryKey: repositoryStatusKey(id),
        queryFn: async () => {
            // On first fetch, get status immediately (wait=false)
            // On subsequent refetches, use long polling (wait=true)
            const shouldWait = initialFetchCompleted.current;
            const result = await apiClient.getRepositoryStatus(id!, shouldWait);
            initialFetchCompleted.current = true;
            return result;
        },
        enabled: !!id,
        refetchInterval: 1, // Long polling: refetch immediately after response (server waits up to 30s)
        refetchIntervalInBackground: true, // Keep polling even when tab is not focused
        retry: true, // Retry on errors (immediate reconnect)
        retryDelay: 0, // No delay between retries (immediate reconnect)
    });
};

export const useFileTree = (id: string | undefined) => {
    return useQuery({
        queryKey: ['file-tree', id],
        queryFn: () => apiClient.getFileTree(id!),
        enabled: !!id,
    });
};

export const useFileContent = (id: string | undefined, filePath: string | undefined) => {
    return useQuery({
        queryKey: ['file-content', id, filePath],
        queryFn: () => apiClient.getFileContent(id!, filePath!),
        enabled: !!id && !!filePath,
    });
};

export const useCommitHistory = (id: string | undefined, limit = 50) => {
    return useQuery({
        queryKey: ['commit-history', id, limit],
        queryFn: () => apiClient.getCommitHistory(id!, limit),
        enabled: !!id,
    });
};

export const useCommitDetails = (id: string | undefined, commitHash: string | undefined) => {
    return useQuery({
        queryKey: ['commit-details', id, commitHash],
        queryFn: () => apiClient.getCommitDetails(id!, commitHash!),
        enabled: !!id && !!commitHash,
    });
};

export const useBranches = (id: string | undefined) => {
    return useQuery({
        queryKey: ['branches', id],
        queryFn: () => apiClient.getBranches(id!),
        enabled: !!id,
    });
};

export const useCreateBranch = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ repositoryId, branchName }: { repositoryId: string; branchName: string }) =>
            apiClient.createBranch(repositoryId, branchName),
        onSuccess: async (_data, variables) => {
            queryClient.invalidateQueries({ queryKey: ['branches', variables.repositoryId] });
            queryClient.invalidateQueries({ queryKey: ['repository-status', variables.repositoryId] });
            await triggerRepositoryStatus(queryClient, variables.repositoryId);
        },
    });
};

export const useSwitchBranch = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ repositoryId, branchName }: { repositoryId: string; branchName: string }) =>
            apiClient.switchBranch(repositoryId, branchName),
        onSuccess: async (_data, variables) => {
            queryClient.invalidateQueries({ queryKey: ['branches', variables.repositoryId] });
            queryClient.invalidateQueries({ queryKey: ['repository-status', variables.repositoryId] });
            queryClient.invalidateQueries({ queryKey: ['repository', variables.repositoryId] });
            await triggerRepositoryStatus(queryClient, variables.repositoryId);
        },
    });
};

export const useDeleteBranch = () => {
    const queryClient = useQueryClient();
    
    return useMutation({
        mutationFn: ({ repositoryId, branchName }: { repositoryId: string; branchName: string }) =>
            apiClient.deleteBranch(repositoryId, branchName),
        onSuccess: async (_data, variables) => {
            queryClient.invalidateQueries({ queryKey: ['branches', variables.repositoryId] });
            queryClient.invalidateQueries({ queryKey: ['repository-status', variables.repositoryId] });
            await triggerRepositoryStatus(queryClient, variables.repositoryId);
        },
    });
};

export const useStageFile = () => {
    const queryClient = useQueryClient();
    
    return useMutation({
        mutationFn: ({ repositoryId, filePath }: { repositoryId: string; filePath: string }) =>
            apiClient.stageFile(repositoryId, filePath),
        onSuccess: async (_data, variables) => {
            queryClient.invalidateQueries({ queryKey: ['repository-status', variables.repositoryId] });
            await triggerRepositoryStatus(queryClient, variables.repositoryId);
        },
    });
};

export const useUnstageFile = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ repositoryId, filePath }: { repositoryId: string; filePath: string }) =>
            apiClient.unstageFile(repositoryId, filePath),
        onSuccess: async (_data, variables) => {
            queryClient.invalidateQueries({ queryKey: ['repository-status', variables.repositoryId] });
            await triggerRepositoryStatus(queryClient, variables.repositoryId);
        },
    });
};

export const usePush = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ repositoryId, force }: { repositoryId: string; force?: boolean }) =>
            force ? apiClient.forcePush(repositoryId) : apiClient.push(repositoryId),
        onSuccess: async (_data, variables) => {
            queryClient.invalidateQueries({ queryKey: ['repository-status', variables.repositoryId] });
            queryClient.invalidateQueries({ queryKey: ['commit-history', variables.repositoryId] });
            await triggerRepositoryStatus(queryClient, variables.repositoryId);
        },
    });
};
