import { atom } from 'jotai';

export type ActiveView = 'files' | 'history' | 'branches' | 'settings' | 'status';
export type Theme = 'light' | 'dark' | 'system';

// Layout state
export const sidebarOpenAtom = atom<boolean>(true);
export const sidebarWidthAtom = atom<number>(280);

// View state
export const activeViewAtom = atom<ActiveView>('files');
export const themeAtom = atom<Theme>('system');

// File selection and search
export const selectedFilesAtom = atom<string[]>([]);
export const searchQueryAtom = atom<string>('');

// Modal and dialog state
export const showSettingsDialogAtom = atom<boolean>(false);
export const showCommitDialogAtom = atom<boolean>(false);
export const showBranchDialogAtom = atom<boolean>(false);

// Loading and error states
export const globalLoadingAtom = atom<boolean>(false);
export const globalErrorAtom = atom<string | null>(null);

// Toast notifications
export interface ToastMessage {
  id: string;
  message: string;
  type: 'success' | 'error' | 'warning' | 'info';
  duration?: number;
}

export const toastMessagesAtom = atom<ToastMessage[]>([]);

// Status bar
export const statusMessageAtom = atom<string>('Ready');
export const progressAtom = atom<{ current: number; total: number } | null>(null);