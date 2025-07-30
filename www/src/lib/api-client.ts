import type {
    Repository,
    RepositoryStatus,
    Branch,
    Commit,
    CommitDetail,
    FileInfo,
    CreateRepositoryRequest,
    CommitRequest,
    DirectoryListing,
    DirectoryEntry,
} from '../types/api';

const API_BASE_URL = import.meta.env.VITE_API_BASE || 'http://localhost:8080';

class ApiClient {
    private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
        const url = `${API_BASE_URL}${endpoint}`;
        console.log('API Client - Making request to:', url);

        const config: RequestInit = {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
            ...options,
        };

        try {
            const response = await fetch(url, config);
            console.log('API Client - Response status:', response.status);

            if (!response.ok) {
                const errorText = await response.text();
                console.error('API Client - Error response:', errorText);
                throw new ApiError({
                    message: errorText || `HTTP ${response.status}: ${response.statusText}`,
                    status: response.status,
                });
            }

            if (response.status === 204) {
                return {} as T;
            }

            const data = await response.json();
            console.log('API Client - Response data:', data);
            return data;
        } catch (error) {
            console.error('API Client - Request failed:', error);
            if (error instanceof ApiError) {
                throw error;
            }
            throw new ApiError({
                message: error instanceof Error ? error.message : 'Network error',
                status: 0,
            });
        }
    }

    // Repository management
    async getRepositories(): Promise<Repository[]> {
        return this.request<Repository[]>('/repos');
    }

    async getRepository(id: string): Promise<Repository> {
        return this.request<Repository>(`/repos/${id}`);
    }

    async createRepository(data: CreateRepositoryRequest): Promise<Repository> {
        return this.request<Repository>('/repos', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }

    async importRepository(data: { path: string; name?: string }): Promise<Repository> {
        return this.request<Repository>('/repos/import', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }

    async deleteRepository(id: string): Promise<void> {
        return this.request<void>(`/repos/${id}`, {
            method: 'DELETE',
        });
    }

    // Repository status and Git operations
    async getRepositoryStatus(id: string): Promise<RepositoryStatus> {
        return this.request<RepositoryStatus>(`/repos/${id}/status`);
    }

    async getCommitHistory(id: string, limit = 50): Promise<Commit[]> {
        return this.request<Commit[]>(`/repos/${id}/commits?limit=${limit}`);
    }

    async getCommitDetails(id: string, commitHash: string): Promise<CommitDetail> {
        return this.request<CommitDetail>(`/repos/${id}/commits/${commitHash}`);
    }

    async getBranches(id: string): Promise<Branch[]> {
        return this.request<Branch[]>(`/repos/${id}/branches`);
    }

    async createCommit(id: string, data: CommitRequest): Promise<void> {
        return this.request<void>(`/repos/${id}/commit`, {
            method: 'POST',
            body: JSON.stringify(data),
        });
    }

    async createBranch(id: string, name: string): Promise<void> {
        return this.request<void>(`/repos/${id}/branches`, {
            method: 'POST',
            body: JSON.stringify({ name }),
        });
    }

    async switchBranch(id: string, branch: string): Promise<void> {
        return this.request<void>(`/repos/${id}/branches/${branch}`, {
            method: 'PUT',
        });
    }

    async deleteBranch(id: string, branch: string): Promise<void> {
        return this.request<void>(`/repos/${id}/branches/${branch}`, {
            method: 'DELETE',
        });
    }

    async push(id: string): Promise<void> {
        return this.request<void>(`/repos/${id}/push`, {
            method: 'POST',
        });
    }

    async pull(id: string): Promise<void> {
        return this.request<void>(`/repos/${id}/pull`, {
            method: 'POST',
        });
    }

    async stageFile(id: string, filePath: string): Promise<void> {
        return this.request<void>(`/repos/${id}/stage/${filePath}`, {
            method: 'POST',
        });
    }

    async unstageFile(id: string, filePath: string): Promise<void> {
        return this.request<void>(`/repos/${id}/stage/${filePath}`, {
            method: 'DELETE',
        });
    }

    // File operations
    async getFileTree(id: string): Promise<FileInfo[]> {
        return this.request<FileInfo[]>(`/repos/${id}/files`);
    }

    async getFileContent(id: string, filePath: string): Promise<string> {
        const response = await fetch(`${API_BASE_URL}/repos/${id}/files/${filePath}`);
        if (!response.ok) {
            throw new ApiError({
                message: `Failed to fetch file: ${response.statusText}`,
                status: response.status,
            });
        }
        return response.text();
    }

    async saveFileContent(id: string, filePath: string, content: string): Promise<void> {
        return this.request<void>(`/repos/${id}/files/${filePath}`, {
            method: 'PUT',
            body: content,
            headers: {
                'Content-Type': 'text/plain',
            },
        });
    }

    // Filesystem browsing
    async browseDirectory(path?: string): Promise<DirectoryListing> {
        const url = path
            ? `/filesystem/browse?path=${encodeURIComponent(path)}`
            : '/filesystem/browse';
        return this.request<DirectoryListing>(url);
    }

    async getVolumeRoots(): Promise<{ roots: DirectoryEntry[] }> {
        return this.request<{ roots: DirectoryEntry[] }>('/filesystem/roots');
    }
}

// Create API client instance
export const apiClient = new ApiClient();

// Custom error class
class ApiError extends Error {
    details: { message: string; status: number };

    constructor(details: { message: string; status: number }) {
        super(details.message);
        this.name = 'ApiError';
        this.details = details;
    }
}
