import { Link, useLocation } from "wouter";
import { 
  LayoutDashboard, 
  Settings, 
  BrainCircuit, 
  Activity, 
  Terminal,
  Cpu
} from "lucide-react";
import { cn } from "@/lib/utils";

export function Layout({ children }: { children: React.ReactNode }) {
  const [location] = useLocation();

  const navItems = [
    { href: "/", icon: LayoutDashboard, label: "OVERVIEW" },
    { href: "/memory", icon: BrainCircuit, label: "AGENT MEMORY" },
    { href: "/logs", icon: Terminal, label: "SYSTEM LOGS" },
    { href: "/config", icon: Settings, label: "CONFIGURATION" },
  ];

  return (
    <div className="flex h-screen w-full bg-background text-foreground overflow-hidden font-mono scanline">
      {/* Sidebar */}
      <aside className="w-64 border-r border-border bg-card/50 flex flex-col">
        <div className="p-6 border-b border-border">
          <div className="flex items-center gap-2 text-primary">
            <Cpu className="h-6 w-6" />
            <h1 className="text-lg font-bold tracking-wider">WATCHDOG</h1>
          </div>
          <p className="text-xs text-muted-foreground mt-1">V.2.0.4 [STABLE]</p>
        </div>

        <nav className="flex-1 p-4 space-y-2">
          {navItems.map((item) => (
            <Link key={item.href} href={item.href}>
              <a className={cn(
                "flex items-center gap-3 px-3 py-2 text-sm transition-colors hover:text-primary hover:bg-white/5",
                location === item.href ? "text-primary bg-primary/10 border-l-2 border-primary" : "text-muted-foreground border-l-2 border-transparent"
              )}>
                <item.icon className="h-4 w-4" />
                {item.label}
              </a>
            </Link>
          ))}
        </nav>

        <div className="p-4 border-t border-border">
          <div className="flex items-center gap-2 mb-2">
            <div className="h-2 w-2 rounded-full bg-green-500 animate-pulse" />
            <span className="text-xs text-green-500 font-bold">SYSTEM ONLINE</span>
          </div>
          <div className="text-[10px] text-muted-foreground">
            UPTIME: 42:10:33
            <br />
            MEM: 2.4GB / 16GB
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto bg-background/50 relative">
        <div className="absolute inset-0 pointer-events-none bg-[radial-gradient(circle_at_center,rgba(6,182,212,0.03),transparent_70%)]" />
        {children}
      </main>
    </div>
  );
}
