import { useEffect, useCallback } from 'react';
import { useAtom } from 'jotai';
import {
  vimModeEnabledAtom,
  vimFocusContextAtom,
  vimFocusIndexAtom,
  vimFocusableCountAtom,
  vimLastContentContextAtom,
  type VimFocusContext,
} from '@/store/atoms';

export interface VimNavigationOptions {
  context: VimFocusContext;
  itemCount: number;
  onNavigate?: (index: number) => void;
  onActivate?: (index: number) => void;
  onExpand?: (index: number) => void;
  onCollapse?: (index: number) => void;
  onPanelLeft?: () => void;
  onPanelRight?: () => void;
  enabled?: boolean;
}

/**
 * Hook for vim-style keyboard navigation within a component
 */
export function useVimNavigation(options: VimNavigationOptions) {
  const {
    context,
    itemCount,
    onNavigate,
    onActivate,
    onExpand,
    onCollapse,
    onPanelLeft,
    onPanelRight,
    enabled = true,
  } = options;

  const [vimEnabled] = useAtom(vimModeEnabledAtom);
  const [currentContext, setCurrentContext] = useAtom(vimFocusContextAtom);
  const [focusIndex, setFocusIndex] = useAtom(vimFocusIndexAtom);
  const [, setFocusableCount] = useAtom(vimFocusableCountAtom);
  const [, setLastContentContext] = useAtom(vimLastContentContextAtom);

  const isActive = vimEnabled && enabled && currentContext === context;

  // Track last content context when this context becomes active
  useEffect(() => {
    if (isActive && context !== 'header' && context !== 'none') {
      setLastContentContext(context);
    }
  }, [isActive, context, setLastContentContext]);

  // Update focusable count when it changes
  useEffect(() => {
    if (isActive) {
      setFocusableCount(itemCount);
    }
  }, [isActive, itemCount, setFocusableCount]);

  // Handle navigation within this context
  const handleNavigate = useCallback(
    (direction: 'up' | 'down') => {
      if (!isActive || itemCount === 0) return;

      let newIndex = focusIndex;
      if (direction === 'down') {
        newIndex = Math.min(focusIndex + 1, itemCount - 1);
      } else {
        newIndex = Math.max(focusIndex - 1, 0);
      }

      if (newIndex !== focusIndex) {
        setFocusIndex(newIndex);
        onNavigate?.(newIndex);
      }
    },
    [isActive, itemCount, focusIndex, setFocusIndex, onNavigate]
  );

  // Handle horizontal navigation (expand/collapse or panel switching)
  const handleHorizontal = useCallback(
    (direction: 'left' | 'right') => {
      if (!isActive) return;

      if (direction === 'left') {
        if (onCollapse && context === 'file-tree') {
          onCollapse(focusIndex);
        } else if (onPanelLeft) {
          onPanelLeft();
        }
      } else {
        if (onExpand && context === 'file-tree') {
          onExpand(focusIndex);
        } else if (onPanelRight) {
          onPanelRight();
        }
      }
    },
    [isActive, context, focusIndex, onCollapse, onExpand, onPanelLeft, onPanelRight]
  );

  // Handle item activation
  const handleActivate = useCallback(() => {
    if (!isActive || itemCount === 0) return;
    onActivate?.(focusIndex);
  }, [isActive, itemCount, focusIndex, onActivate]);

  // Set this context as active when vim mode is enabled and component is visible
  const activateContext = useCallback(() => {
    if (vimEnabled && enabled) {
      setCurrentContext(context);
      setFocusIndex(0);
      setFocusableCount(itemCount);
    }
  }, [vimEnabled, enabled, context, itemCount, setCurrentContext, setFocusIndex, setFocusableCount]);

  // Listen for h/l keys when this context is active
  useEffect(() => {
    if (!isActive) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'h' && !e.shiftKey && !e.ctrlKey && !e.metaKey && !e.altKey) {
        e.preventDefault();
        handleHorizontal('left');
      } else if (e.key === 'l' && !e.shiftKey && !e.ctrlKey && !e.metaKey && !e.altKey) {
        e.preventDefault();
        handleHorizontal('right');
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isActive, handleHorizontal]);

  return {
    isVimActive: isActive,
    currentIndex: focusIndex,
    navigate: handleNavigate,
    horizontal: handleHorizontal,
    activate: handleActivate,
    activateContext,
  };
}

/**
 * Global hook for managing vim mode (should be used in AppLayout or similar top-level component)
 */
export function useGlobalVimMode() {
  const [vimEnabled, setVimEnabled] = useAtom(vimModeEnabledAtom);
  const [currentContext, setCurrentContext] = useAtom(vimFocusContextAtom);
  const [focusIndex, setFocusIndex] = useAtom(vimFocusIndexAtom);
  const [focusableCount] = useAtom(vimFocusableCountAtom);
  const [lastContentContext] = useAtom(vimLastContentContextAtom);

  // Handle global keyboard events
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't trigger if user is typing in an input/textarea
      const target = e.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
        return;
      }

      // Toggle vim mode with 'v' key
      if (e.key === 'v' && !e.shiftKey && !e.ctrlKey && !e.metaKey && !e.altKey) {
        e.preventDefault();
        setVimEnabled(!vimEnabled);
        if (!vimEnabled) {
          // When enabling, start in header context
          setCurrentContext('header');
          setFocusIndex(0);
        } else {
          // When disabling, clear context
          setCurrentContext('none');
        }
        return;
      }

      // Only handle vim commands when vim mode is enabled
      if (!vimEnabled) return;

      // Escape key - exit vim mode
      if (e.key === 'Escape') {
        e.preventDefault();
        setVimEnabled(false);
        setCurrentContext('none');
        return;
      }

      // Tab key - switch between header and content
      if (e.key === 'Tab') {
        e.preventDefault();
        if (currentContext === 'header') {
          // Switch to last content context
          setCurrentContext(lastContentContext);
          setFocusIndex(0);
        } else {
          // Switch to header
          setCurrentContext('header');
          setFocusIndex(0);
        }
        return;
      }

      // Shift + H/L for header navigation (left/right)
      if (e.shiftKey && (e.key === 'H' || e.key === 'L')) {
        e.preventDefault();
        if (currentContext !== 'header') {
          setCurrentContext('header');
          setFocusIndex(0);
        } else {
          const newIndex =
            e.key === 'L'
              ? Math.min(focusIndex + 1, focusableCount - 1)
              : Math.max(focusIndex - 1, 0);
          setFocusIndex(newIndex);
        }
        return;
      }

      // j/k for up/down navigation (lowercase only)
      if (e.key === 'j' && !e.shiftKey) {
        e.preventDefault();
        if (currentContext !== 'header' && focusIndex < focusableCount - 1) {
          setFocusIndex(focusIndex + 1);
        }
        return;
      }

      if (e.key === 'k' && !e.shiftKey) {
        e.preventDefault();
        if (currentContext !== 'header' && focusIndex > 0) {
          setFocusIndex(focusIndex - 1);
        }
        return;
      }

      // h/l for horizontal navigation
      if (e.key === 'h' || e.key === 'l') {
        e.preventDefault();
        // These will be handled by individual component hooks
        // We just prevent default here
        return;
      }

      // Enter to activate
      if (e.key === 'Enter' && vimEnabled) {
        e.preventDefault();
        // Activation will be handled by individual component hooks
        return;
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [vimEnabled, currentContext, focusIndex, focusableCount, lastContentContext, setVimEnabled, setCurrentContext, setFocusIndex]);

  const toggleVimMode = useCallback(() => {
    setVimEnabled(!vimEnabled);
    if (!vimEnabled) {
      setCurrentContext('header');
      setFocusIndex(0);
    } else {
      setCurrentContext('none');
    }
  }, [vimEnabled, setVimEnabled, setCurrentContext, setFocusIndex]);

  return {
    vimEnabled,
    currentContext,
    focusIndex,
    toggleVimMode,
  };
}
