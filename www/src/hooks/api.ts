import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../lib/api-client';
import type { CreateRepositoryRequest, CommitRequest } from '../types/api';

// Query keys
export const queryKeys = {
    repositories: ['repositories'] as const,
    repository: (id: string) => ['repository', id] as const,
    repositoryStatus: (id: string) => ['repository', id, 'status'] as const,
    commitHistory: (id: string, limit?: number) => ['repository', id, 'commits', limit] as const,
    branches: (id: string) => ['repository', id, 'branches'] as const,
    fileTree: (id: string) => ['repository', id, 'files'] as const,
    fileContent: (id: string, path: string) => ['repository', id, 'file', path] as const,
    directoryBrowse: (path?: string) => ['filesystem', 'browse', path] as const,
    volumeRoots: ['filesystem', 'roots'] as const,
};

// Repository queries
export const useRepositories = () => {
    return useQuery({
        queryKey: queryKeys.repositories,
        queryFn: () => apiClient.getRepositories(),
    });
};

export const useRepository = (id: string) => {
    return useQuery({
        queryKey: queryKeys.repository(id),
        queryFn: () => apiClient.getRepository(id),
        enabled: !!id,
    });
};

export const useRepositoryStatus = (id: string) => {
    return useQuery({
        queryKey: queryKeys.repositoryStatus(id),
        queryFn: () => apiClient.getRepositoryStatus(id),
        enabled: !!id,
        refetchInterval: 5000, // Refetch every 5 seconds for live updates
    });
};

export const useCommitHistory = (id: string, limit = 50) => {
    return useQuery({
        queryKey: queryKeys.commitHistory(id, limit),
        queryFn: () => apiClient.getCommitHistory(id, limit),
        enabled: !!id,
    });
};

export const useBranches = (id: string) => {
    return useQuery({
        queryKey: queryKeys.branches(id),
        queryFn: () => apiClient.getBranches(id),
        enabled: !!id,
    });
};

export const useFileTree = (id: string) => {
    return useQuery({
        queryKey: queryKeys.fileTree(id),
        queryFn: () => apiClient.getFileTree(id),
        enabled: !!id,
    });
};

export const useFileContent = (id: string, filePath: string) => {
    return useQuery({
        queryKey: queryKeys.fileContent(id, filePath),
        queryFn: () => apiClient.getFileContent(id, filePath),
        enabled: !!id && !!filePath,
    });
};

// Filesystem queries
export const useDirectoryBrowse = (path?: string) => {
    return useQuery({
        queryKey: queryKeys.directoryBrowse(path),
        queryFn: () => apiClient.browseDirectory(path),
    });
};

export const useVolumeRoots = () => {
    return useQuery({
        queryKey: queryKeys.volumeRoots,
        queryFn: () => apiClient.getVolumeRoots(),
    });
};

// Repository mutations
export const useCreateRepository = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateRepositoryRequest) => apiClient.createRepository(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.repositories });
        },
    });
};

export const useImportRepository = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: { path: string; name?: string }) => apiClient.importRepository(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.repositories });
        },
    });
};

export const useDeleteRepository = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => apiClient.deleteRepository(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.repositories });
        },
    });
};

export const useCreateCommit = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: CommitRequest }) =>
            apiClient.createCommit(id, data),
        onSuccess: (_, { id }) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.repositoryStatus(id) });
            queryClient.invalidateQueries({ queryKey: queryKeys.commitHistory(id) });
        },
    });
};

export const useCreateBranch = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, name }: { id: string; name: string }) =>
            apiClient.createBranch(id, name),
        onSuccess: (_, { id }) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.branches(id) });
        },
    });
};

export const useSwitchBranch = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, branch }: { id: string; branch: string }) =>
            apiClient.switchBranch(id, branch),
        onSuccess: (_, { id }) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.repository(id) });
            queryClient.invalidateQueries({ queryKey: queryKeys.repositoryStatus(id) });
            queryClient.invalidateQueries({ queryKey: queryKeys.fileTree(id) });
        },
    });
};

export const usePush = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => apiClient.push(id),
        onSuccess: (_, id) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.repositoryStatus(id) });
        },
    });
};

export const usePull = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => apiClient.pull(id),
        onSuccess: (_, id) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.repository(id) });
            queryClient.invalidateQueries({ queryKey: queryKeys.repositoryStatus(id) });
            queryClient.invalidateQueries({ queryKey: queryKeys.fileTree(id) });
            queryClient.invalidateQueries({ queryKey: queryKeys.commitHistory(id) });
        },
    });
};

export const useSaveFile = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            id,
            filePath,
            content,
        }: {
            id: string;
            filePath: string;
            content: string;
        }) => apiClient.saveFileContent(id, filePath, content),
        onSuccess: (_, { id, filePath }) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.fileContent(id, filePath) });
            queryClient.invalidateQueries({ queryKey: queryKeys.repositoryStatus(id) });
        },
    });
};
