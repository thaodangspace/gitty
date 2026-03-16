import { useEffect, useState } from 'react';
import { useAtom } from 'jotai';
import { themeAtom, Theme } from '@/store/atoms/ui-atoms';

const THEME_STORAGE_KEY = 'gitty-theme';

function getSystemTheme(): Theme {
    if (typeof window === 'undefined') return 'light';
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyTheme(theme: Theme) {
    const root = document.documentElement;
    const actualTheme = theme === 'system' ? getSystemTheme() : theme;
    
    if (actualTheme === 'dark') {
        root.classList.add('dark');
    } else {
        root.classList.remove('dark');
    }
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
    const [theme, setTheme] = useAtom(themeAtom);
    const [isInitialized, setIsInitialized] = useState(false);

    useEffect(() => {
        if (typeof window !== 'undefined') {
            const storedTheme = localStorage.getItem(THEME_STORAGE_KEY) as Theme | null;
            const initialTheme = storedTheme || 'system';
            setTheme(initialTheme);
            applyTheme(initialTheme);
            setIsInitialized(true);
        }
    }, [setTheme]);

    useEffect(() => {
        if (!isInitialized) return;

        localStorage.setItem(THEME_STORAGE_KEY, theme);
        applyTheme(theme);
    }, [theme, isInitialized]);

    useEffect(() => {
        if (!isInitialized) return;

        const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
        const handleSystemThemeChange = () => {
            if (theme === 'system') {
                applyTheme('system');
            }
        };

        mediaQuery.addEventListener('change', handleSystemThemeChange);
        return () => mediaQuery.removeEventListener('change', handleSystemThemeChange);
    }, [theme, isInitialized]);



    if (!isInitialized) {
        return <>{children}</>;
    }

    return (
        <div data-theme={theme}>
            {children}
        </div>
    );
}

export { useAtom as useTheme, Theme };
