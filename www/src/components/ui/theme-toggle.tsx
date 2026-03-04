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
        if (theme === 'dark') return <Moon className="h-4 w-4" />;
        if (theme === 'light') return <Sun className="h-4 w-4" />;
        return <Monitor className="h-4 w-4" />;
    };

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-9 w-9">
                    {getCurrentIcon()}
                    <span className="sr-only">Toggle theme</span>
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
                {themeOptions.map((option) => {
                    const Icon = option.icon;
                    return (
                        <DropdownMenuItem
                            key={option.value}
                            onClick={() => setTheme(option.value)}
                            className={`gap-2 ${theme === option.value ? 'bg-accent' : ''}`}
                        >
                            <Icon className="h-4 w-4" />
                            {option.label}
                        </DropdownMenuItem>
                    );
                })}
            </DropdownMenuContent>
        </DropdownMenu>
    );
}