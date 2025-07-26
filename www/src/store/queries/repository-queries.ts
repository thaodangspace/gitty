import { useQuery } from '@tanstack/react-query';

export interface Repository {
    id: string;
    name: string;
    path: string;
    status: 'clean' | 'dirty' | 'unknown';
    currentBranch: string;
    lastCommit?: string;
}

const API_BASE = import.meta.env.VITE_API_BASE;

export const useRepositories = () => {
    return useQuery({
        queryKey: ['repositories'],
        queryFn: async (): Promise<Repository[]> => {
            const response = await fetch(`${API_BASE}/repos`);
            if (!response.ok) {
                throw new Error('Failed to fetch repositories');
            }
            return response.json();
        },
    });
};

export const useRepository = (id: string) => {
    return useQuery({
        queryKey: ['repository', id],
        queryFn: async (): Promise<Repository> => {
            const response = await fetch(`${API_BASE}/repos/${id}`);
            if (!response.ok) {
                throw new Error('Failed to fetch repository');
            }
            return response.json();
        },
        enabled: !!id,
    });
};
