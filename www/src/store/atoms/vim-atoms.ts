import { atom } from 'jotai';

export type VimFocusContext =
  | 'header'
  | 'commit-list'
  | 'branch-list'
  | 'file-changes'
  | 'file-tree'
  | 'none';

export interface VimFocusableElement {
  id: string;
  element: HTMLElement;
  context: VimFocusContext;
}

// Main vim mode state
export const vimModeEnabledAtom = atom<boolean>(false);

// Current focus context (which area of the UI has vim focus)
export const vimFocusContextAtom = atom<VimFocusContext>('none');

// Last content context (used for Tab switching)
export const vimLastContentContextAtom = atom<VimFocusContext>('commit-list');

// Index of currently focused item within the context
export const vimFocusIndexAtom = atom<number>(0);

// Track total focusable items in current context
export const vimFocusableCountAtom = atom<number>(0);

// Derived atom to get focus info as a string (for status bar)
export const vimFocusInfoAtom = atom((get) => {
  const enabled = get(vimModeEnabledAtom);
  const context = get(vimFocusContextAtom);
  const index = get(vimFocusIndexAtom);
  const count = get(vimFocusableCountAtom);

  if (!enabled) return null;

  const contextLabels: Record<VimFocusContext, string> = {
    'header': 'Header',
    'commit-list': 'Commits',
    'branch-list': 'Branches',
    'file-changes': 'Changes',
    'file-tree': 'Files',
    'none': 'None',
  };

  const label = contextLabels[context];
  return count > 0 ? `${label} [${index + 1}/${count}]` : label;
});
