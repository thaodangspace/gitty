import { useAtom } from 'jotai';
import { sidebarOpenAtom } from '@/store/atoms';
import { Button } from '@/components/ui/button';
import { Menu, Smartphone, Monitor, Tablet } from 'lucide-react';

export default function MobileTest() {
  const [sidebarOpen, setSidebarOpen] = useAtom(sidebarOpenAtom);

  return (
    <div className="p-4 space-y-4">
      <div className="flex items-center gap-2 mb-4">
        <Smartphone className="h-4 w-4" />
        <span className="text-sm font-medium">Mobile Responsiveness Test</span>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="p-4 border rounded-lg">
          <div className="flex items-center gap-2 mb-2">
            <Smartphone className="h-4 w-4" />
            <span className="text-sm font-medium">Mobile</span>
          </div>
          <p className="text-xs text-muted-foreground">
            Test touch targets and responsive layout on mobile devices.
          </p>
        </div>
        
        <div className="p-4 border rounded-lg">
          <div className="flex items-center gap-2 mb-2">
            <Tablet className="h-4 w-4" />
            <span className="text-sm font-medium">Tablet</span>
          </div>
          <p className="text-xs text-muted-foreground">
            Verify layout adapts properly for tablet screens.
          </p>
        </div>
        
        <div className="p-4 border rounded-lg">
          <div className="flex items-center gap-2 mb-2">
            <Monitor className="h-4 w-4" />
            <span className="text-sm font-medium">Desktop</span>
          </div>
          <p className="text-xs text-muted-foreground">
            Ensure desktop experience remains optimal.
          </p>
        </div>
      </div>
      
      <div className="space-y-2">
        <h3 className="text-sm font-medium">Touch Target Test</h3>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" className="touch-target">Small Button</Button>
          <Button className="touch-target">Regular Button</Button>
          <Button size="lg" className="touch-target">Large Button</Button>
        </div>
      </div>
      
      <div className="space-y-2">
        <h3 className="text-sm font-medium">Responsive Text Test</h3>
        <p className="text-xs md:text-sm lg:text-base">
          This text should scale appropriately across different screen sizes.
        </p>
      </div>
      
      <div className="space-y-2">
        <h3 className="text-sm font-medium">Sidebar Test</h3>
        <Button 
          onClick={() => setSidebarOpen(!sidebarOpen)}
          className="touch-target"
        >
          <Menu className="h-4 w-4 mr-2" />
          Toggle Sidebar
        </Button>
      </div>
    </div>
  );
}