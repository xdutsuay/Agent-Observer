import { Link, useLocation } from "wouter";
import { useQuery } from "@tanstack/react-query";
import {
  LayoutDashboard,
  Settings,
  BrainCircuit,
  Terminal,
  Cpu,
  FolderOpen,
  Search,
  BarChart3,
  Clock,
  Plug,
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
    { href: "/", icon: LayoutDashboard, label: "DASHBOARD" },
    { href: "/projects", icon: FolderOpen, label: "PROJECTS" },
    { href: "/memory", icon: BrainCircuit, label: "MEMORY" },
    { href: "/search", icon: Search, label: "SEARCH" },
    { href: "/patterns", icon: BarChart3, label: "PATTERNS" },
    { href: "/timeline", icon: Clock, label: "TIMELINE" },
    { href: "/usage", icon: Plug, label: "AGENT USAGE" },
    { href: "/logs", icon: Terminal, label: "LOGS" },
    { href: "/config", icon: Settings, label: "CONFIG" },
  ];

  return (
    <div className="flex h-screen w-full bg-[#0a0b0d] text-foreground overflow-hidden font-mono">
      <aside className="w-56 border-r border-[#21262d] bg-[#0a0b0d] flex flex-col">
        <div className="p-5">
          <div className="flex items-center gap-2 mb-1">
            <div className="p-1.5 bg-primary/10 rounded">
              <Cpu className="h-4 w-4 text-primary" />
            </div>
            <div className="font-black text-sm tracking-tighter">AGENT MEMORY</div>
          </div>
          <div className="text-[10px] text-muted-foreground opacity-70">
            v0.4.0 — MCP + HTTP + WS
          </div>
        </div>

        <nav className="flex-1 mt-2 overflow-y-auto">
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 px-5 py-3 text-[11px] font-bold transition-all duration-200 border-l-2",
                location === item.href
                  ? "bg-[#161b22] border-primary text-white"
                  : "border-transparent text-muted-foreground hover:text-white hover:bg-white/5"
              )}
            >
              <item.icon className={cn("h-3.5 w-3.5", location === item.href ? "text-primary" : "")} />
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="p-5 border-t border-[#21262d] bg-[#0d0f14]">
          <div className="flex items-center gap-2 mb-2">
            <div className={cn(
              "h-2 w-2 rounded-full",
              watcherOn ? "bg-primary animate-pulse shadow-[0_0_8px_rgba(0,229,255,0.5)]" : "bg-secondary"
            )} />
            <span className={cn(
              "text-[10px] font-bold tracking-widest",
              watcherOn ? "text-primary" : "text-secondary"
            )}>
              {watcherOn ? "WATCHER ON" : "WATCHER OFF"}
            </span>
          </div>
          <div className="text-[9px] text-muted-foreground leading-relaxed">
            MEM: {metrics?.memory_used_gb?.toFixed(1) ?? "—"} / {metrics?.memory_total_gb?.toFixed(1) ?? "—"} GB
          </div>
        </div>
      </aside>

      <main className="flex-1 overflow-auto bg-[#0a0b0d]">
        {children}
      </main>
    </div>
  );
}
