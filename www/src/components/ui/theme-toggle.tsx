import { useAtom } from 'jotai';
import { themeAtom, Theme } from '@/store/atoms/ui-atoms';
import { Button } from '@/components/ui/button';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Moon, Sun, Monitor } from 'lucide-react';

type ThemeIcon = typeof Sun | typeof Moon | typeof Monitor;

const themeOptions: Array<{ value: Theme; label: string; icon: ThemeIcon }> = [
    { value: 'light', label: 'Light', icon: Sun },
    { value: 'dark', label: 'Dark', icon: Moon },
    { value: 'system', label: 'System', icon: Monitor },
];

export function ThemeToggle() {
    const [theme, setTheme] = useAtom(themeAtom);

    const getCurrentIcon = () => {
        if (theme === 'dark') return <Moon className="h-5 w-5" />;
        if (theme === 'light') return <Sun className="h-5 w-5" />;
        return <Monitor className="h-5 w-5" />;
    };

    const cycleTheme = () => {
        const currentIndex = themeOptions.findIndex(opt => opt.value === theme);
        const nextIndex = (currentIndex + 1) % themeOptions.length;
        setTheme(themeOptions[nextIndex].value);

        if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
            navigator.vibrate?.(10);
        }
    };

    const getThemeLabel = () => {
        const option = themeOptions.find(opt => opt.value === theme);
        return option ? option.label : 'Theme';
    };

    return (
        <>
            {/* Primary touch-friendly button - cycles theme on tap */}
            <Button
                variant="ghost"
                size="icon"
                className="h-11 w-11 min-h-[44px] min-w-[44px] touch-manipulation active:scale-95 transition-transform"
                onClick={cycleTheme}
                onTouchStart={(e) => {
                    // Add visual feedback on touch start for immediate response
                    e.currentTarget.style.transform = 'scale(0.95)';
                }}
                onTouchEnd={(e) => {
                    e.currentTarget.style.transform = 'scale(1)';
                }}
            >
                {getCurrentIcon()}
                <span className="sr-only">Toggle theme (current: {getThemeLabel()})</span>
            </Button>

            {/* Dropdown menu for explicit theme selection - hidden on small screens, available for desktop */}
            <DropdownMenu>
                <DropdownMenuTrigger asChild>
                    <Button
                        variant="ghost"
                        size="icon"
                        className="h-11 w-6 min-h-[44px] min-w-[24px] touch-manipulation active:scale-95 transition-transform md:flex hidden"
                        aria-label="Open theme menu"
                    >
                        <div className="h-5 w-px bg-border" />
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48">
                    {themeOptions.map((option) => {
                        const Icon = option.icon;
                        return (
                            <DropdownMenuItem
                                key={option.value}
                                onClick={() => setTheme(option.value)}
                                className={`gap-2 h-11 touch-manipulation ${theme === option.value ? 'bg-accent' : ''}`}
                            >
                                <Icon className="h-5 w-5" />
                                <span className="text-sm">{option.label}</span>
                            </DropdownMenuItem>
                        );
                    })}
                </DropdownMenuContent>
            </DropdownMenu>
        </>
    );
}