import { atom } from 'jotai';
import type { Repository, Branch, Commit, FileChange } from '../../types/api';

// Repository management
export const repositoriesAtom = atom<Repository[]>([]);
export const selectedRepositoryAtom = atom<Repository | null>(null);
export const selectedRepositoryIdAtom = atom<string | null>(null);
export const isLoadingRepositoriesAtom = atom<boolean>(false);

// Derived atoms
export const selectedRepositoryFromListAtom = atom((get) => {
    const repositories = get(repositoriesAtom);
    const selectedId = get(selectedRepositoryIdAtom);
    return repositories.find((repo) => repo.id === selectedId) || null;
});

// File management
export const selectedFileAtom = atom<string | null>(null);
export const activeFilePathAtom = atom<string | null>(null);
export const fileContentAtom = atom<string>('');

// Git state
export const stagingAreaAtom = atom<FileChange[]>([]);
export const branchesAtom = atom<Branch[]>([]);
export const commitsAtom = atom<Commit[]>([]);
export const currentBranchAtom = atom<string>('');

// File tree state
export const expandedDirectoriesAtom = atom<Set<string>>(new Set());
export const fileTreeAtom = atom<any[]>([]);

// UI state for repository operations
export const isCreatingRepositoryAtom = atom<boolean>(false);
export const showCreateRepositoryDialogAtom = atom<boolean>(false);
export const showFolderSelectionDialogAtom = atom<boolean>(false);

// Directory browsing state
export const currentDirectoryAtom = atom<string>('');
export const directoryHistoryAtom = atom<string[]>([]);
