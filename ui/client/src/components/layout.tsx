import { Link, useLocation } from "wouter";
import { useQuery } from "@tanstack/react-query";
import {
  LayoutDashboard,
  Settings,
  BrainCircuit,
  Terminal,
  Cpu,
  Grid
} from "lucide-react";
import { cn } from "@/lib/utils";

export function Layout({ children }: { children: React.ReactNode }) {
  const [location] = useLocation();
  const { data: status } = useQuery<{ running: boolean }>({
    queryKey: ["/api/status"],
    refetchInterval: 3000,
  });
  const { data: metrics } = useQuery<{
    memory_used_gb: number;
    memory_total_gb: number;
  }>({
    queryKey: ["/api/metrics"],
    refetchInterval: 3000,
  });
  const watcherOn = status?.running ?? false;

  const navItems = [
    { href: "/", icon: Grid, label: "OVERVIEW" },
    { href: "/memory", icon: BrainCircuit, label: "AGENT MEMORY" },
    { href: "/logs", icon: Terminal, label: "SYSTEM LOGS" },
    { href: "/config", icon: Settings, label: "CONFIGURATION" },
  ];

  return (
    <div className="flex h-screen w-full bg-[#0a0b0d] text-foreground overflow-hidden font-mono">
      {/* Sidebar */}
      <aside className="w-64 border-r border-[#21262d] bg-[#0a0b0d] flex flex-col">
        <div className="p-6">
          <div className="flex items-center gap-2 mb-1">
             <div className="p-1 bg-primary/10 rounded">
               <Cpu className="h-5 w-5 text-primary" />
             </div>
             <div className="font-black text-sm tracking-tighter">WATCHDOG</div>
          </div>
          <div className="text-[10px] text-muted-foreground opacity-70">
            V.2.0.4 [STABLE]
          </div>
        </div>

        <nav className="flex-1 mt-4">
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 px-6 py-4 text-xs font-bold transition-all duration-200 border-l-2",
                location === item.href
                  ? "bg-[#161b22] border-primary text-white"
                  : "border-transparent text-muted-foreground hover:text-white hover:bg-white/5"
              )}
            >
              <item.icon className={cn("h-4 w-4", location === item.href ? "text-primary" : "")} />
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="p-6 border-t border-[#21262d] bg-[#0d0f14]">
          <div className="flex items-center gap-2 mb-3">
            <div className={cn(
              "h-2 w-2 rounded-full",
              watcherOn ? "bg-primary animate-pulse shadow-[0_0_8px_rgba(0,229,255,0.5)]" : "bg-secondary"
            )} />
            <span className={cn(
              "text-[10px] font-bold tracking-widest",
              watcherOn ? "text-primary" : "text-secondary"
            )}>
              {watcherOn ? "WATCHER ACTIVE" : "WATCHER OFF"}
            </span>
          </div>
          <div className="text-[10px] text-muted-foreground leading-relaxed">
            MEM: {metrics?.memory_used_gb?.toFixed(1) ?? "—"} GB /{" "}
            {metrics?.memory_total_gb?.toFixed(1) ?? "—"} GB
            <br />
            {watcherOn ? "Watching ~/localcode" : "Run: agent-memory serve"}
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto bg-[#0a0b0d]">
        {children}
      </main>
    </div>
  );
}
