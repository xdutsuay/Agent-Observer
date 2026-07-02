import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAgentSimulation } from "@/lib/simulation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  Cpu,
  Box,
  AlertTriangle,
  FolderOpen,
  Shield,
  Flame,
  Plug,
  HardDrive,
  Sparkles,
  RefreshCw,
  Trash2,
  Zap,
  Eye,
  ThumbsUp,
} from "lucide-react";
import {
  AreaChart,
  Area,
  ResponsiveContainer,
  CartesianGrid,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
} from "recharts";
import { cn } from "@/lib/utils";
import { useState, useEffect } from "react";
import { Link } from "wouter";
import { apiRequest } from "@/lib/queryClient";

interface PatternReport {
  health_score: { score: number; grade: string; reasons: string[] };
  activity_trends: { total: number; by_day: Record<string, number>; by_kind: Record<string, number> };
  common_error_categories: { category: string; count: number }[];
  recurring_failures: any[];
}

interface ProjectInfo {
  repo_id: string;
  name: string;
  language: string | null;
  framework: string | null;
  memory_count: number;
  failure_count: number;
  tags: string[];
}

interface Hotspot {
  repo_id: string;
  path: string;
  unresolved_failures: number;
}

export default function Dashboard() {
  const { events, metrics, isRunning, repos } = useAgentSimulation();
  const [chartData, setChartData] = useState<{ time: number; val: number }[]>([]);
  const qc = useQueryClient();

  const { data: patterns } = useQuery<PatternReport>({
    queryKey: ["/api/patterns"],
    refetchInterval: 10000,
  });

  const { data: projectsData } = useQuery<{ projects: ProjectInfo[] }>({
    queryKey: ["/api/projects"],
    refetchInterval: 10000,
  });

  const { data: hotspotsData } = useQuery<{ hotspots: Hotspot[] }>({
    queryKey: ["/api/hotspots"],
    refetchInterval: 10000,
  });

  const { data: usageSummary } = useQuery<{
    last_24h: number;
    total_interactions: number;
    reads: number;
    writes: number;
    running_ides: { label: string }[];
  }>({
    queryKey: ["/api/usage/summary"],
    refetchInterval: 8000,
  });

  const { data: disk } = useQuery<{
    overall: {
      data_root_bytes_human: string;
      total_memory_attributed_human: string;
      total_workspace_human: string;
    };
    projects: { name: string; memory_store_bytes: number; workspace_bytes: number | null }[];
  }>({
    queryKey: ["/api/disk-usage"],
    refetchInterval: 60000,
  });

  const refreshMutation = useMutation({
    mutationFn: async () => {
      const res = await apiRequest("POST", "/api/v1/relevance/refresh");
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["/api/patterns"] });
      qc.invalidateQueries({ queryKey: ["/api/projects"] });
    },
  });

  useEffect(() => {
    setChartData((prev) => {
      const newData = [...prev, { time: Date.now() / 1000, val: (metrics.fs_events_per_sec || 0) * 10 }];
      return newData.slice(-40);
    });
  }, [metrics.fs_events_per_sec]);

  const health = patterns?.health_score;
  const projects = projectsData?.projects || [];
  const hotspots = hotspotsData?.hotspots || [];
  const errorCategories = patterns?.common_error_categories || [];
  const activityByKind = patterns?.activity_trends?.by_kind || {};

  const kindData = Object.entries(activityByKind).map(([kind, count]) => ({ kind, count }));

  const gradeColor = (grade: string) => {
    if (grade === "A") return "text-green-400";
    if (grade === "B") return "text-blue-400";
    if (grade === "C") return "text-yellow-400";
    if (grade === "D") return "text-orange-400";
    return "text-red-400";
  };

  return (
    <Layout>
      <div className="p-8 space-y-8">
        {/* Header row */}
        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-2xl font-black tracking-tight text-white uppercase">
              Dashboard
            </h1>
            <p className="text-xs text-[#8b949e] mt-1">
              Agent memory system overview — {projects.length} projects tracked
            </p>
          </div>

          <div className="flex items-center gap-3">
            {/* Refresh relevance button */}
            <Button
              variant="outline"
              size="sm"
              className="text-[10px] font-mono gap-1 h-8"
              onClick={() => refreshMutation.mutate()}
              disabled={refreshMutation.isPending}
            >
              <RefreshCw className={cn("h-3 w-3", refreshMutation.isPending && "animate-spin")} />
              {refreshMutation.isPending ? "Refreshing..." : "Refresh Scores"}
            </Button>

            {/* Health badge */}
            {health && (
              <div className="text-right">
                <div className="text-[10px] text-[#8b949e] uppercase tracking-widest mb-1">Health Score</div>
                <div className={cn("text-3xl font-black", gradeColor(health.grade))}>
                  {health.grade}
                  <span className="text-sm ml-1 opacity-60">{health.score}/100</span>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Stat cards */}
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
          <StatCard icon={FolderOpen} label="PROJECTS" value={projects.length} color="text-primary" />
          <StatCard icon={Box} label="MEMORIES" value={patterns?.activity_trends?.total ?? 0} color="text-blue-400" />
          <StatCard icon={AlertTriangle} label="FAILURES" value={hotspots.reduce((s, h) => s + h.unresolved_failures, 0)} color="text-orange-400" />
          <StatCard icon={Activity} label="EVENTS/SEC" value={Math.round(metrics.fs_events_per_sec || 0)} color="text-green-400" />
          <StatCard icon={Sparkles} label="RELEVANCE" value={`${refreshMutation.data?.refreshed ?? "—"}`} color="text-yellow-400" subtitle={refreshMutation.data ? `${refreshMutation.data.noise_classified} noise` : undefined} />
        </div>

        {/* Usage + Disk row */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Link href="/usage">
            <Card className="bg-[#111317] border-[#21262d] hover:border-primary/40 transition-colors cursor-pointer h-full">
              <CardContent className="p-4 flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                  <Plug className="h-5 w-5 text-primary" />
                  <div>
                    <div className="text-[10px] text-[#8b949e] uppercase tracking-widest">Agent usage</div>
                    <div className="text-sm font-bold text-white">
                      {usageSummary?.last_24h ?? 0} calls in 24h
                      <span className="text-[#8b949e] font-normal ml-2">
                        ({usageSummary?.reads ?? 0} reads · {usageSummary?.writes ?? 0} writes)
                      </span>
                    </div>
                  </div>
                </div>
                <div className="flex flex-wrap gap-1 justify-end">
                  {(usageSummary?.running_ides || []).slice(0, 4).map((ide) => (
                    <Badge key={ide.label} variant="outline" className="text-[8px]">
                      {ide.label}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          </Link>

          {disk && (
            <Card className="bg-[#111317] border-[#21262d]">
              <CardContent className="p-4 flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                  <HardDrive className="h-5 w-5 text-primary" />
                  <div>
                    <div className="text-[10px] text-[#8b949e] uppercase tracking-widest">Disk usage</div>
                    <div className="text-sm font-bold text-white">
                      Agent store {disk.overall.data_root_bytes_human}
                      <span className="text-[#8b949e] font-normal ml-2">
                        · workspaces {disk.overall.total_workspace_human}
                      </span>
                    </div>
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  {disk.projects
                    .filter((p) => p.memory_store_bytes > 0 || (p.workspace_bytes ?? 0) > 0)
                    .slice(0, 5)
                    .map((p) => (
                      <Badge key={p.name} variant="outline" className="text-[8px] font-mono">
                        {p.name}
                      </Badge>
                    ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Activity chart */}
          <div className="lg:col-span-8 space-y-3">
            <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.15em] uppercase px-1">
              Activity stream
            </div>
            <Card className="bg-[#111317] border-[#21262d] h-[300px] overflow-hidden">
              <div className="h-full w-full p-4">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={chartData} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
                    <defs>
                      <linearGradient id="colorWave" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#00e5ff" stopOpacity={0.15} />
                        <stop offset="95%" stopColor="#00e5ff" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#21262d" vertical={false} />
                    <Area type="monotone" dataKey="val" stroke="#00e5ff" strokeWidth={2} fillOpacity={1} fill="url(#colorWave)" animationDuration={300} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </Card>
          </div>

          {/* Memory by kind */}
          <div className="lg:col-span-4 space-y-3">
            <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.15em] uppercase px-1">
              Memories by type
            </div>
            <Card className="bg-[#111317] border-[#21262d] h-[300px] overflow-hidden">
              <div className="h-full w-full p-4">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={kindData} layout="vertical" margin={{ top: 5, right: 20, left: 50, bottom: 5 }}>
                    <XAxis type="number" tick={{ fill: "#8b949e", fontSize: 10 }} />
                    <YAxis type="category" dataKey="kind" tick={{ fill: "#8b949e", fontSize: 10 }} width={60} />
                    <Tooltip
                      contentStyle={{ background: "#111317", border: "1px solid #21262d", fontSize: 11 }}
                      labelStyle={{ color: "#fff" }}
                    />
                    <Bar dataKey="count" fill="#00e5ff" radius={[0, 4, 4, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Failure hotspots */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-[10px] text-orange-400 font-bold tracking-[0.15em] uppercase px-1">
              <Flame className="h-3 w-3" /> Failure hotspots
            </div>
            <Card className="bg-[#111317] border-[#21262d]">
              <CardContent className="p-0 divide-y divide-[#21262d]">
                {hotspots.length === 0 ? (
                  <div className="p-8 text-center text-[#8b949e] text-xs">All projects healthy</div>
                ) : (
                  hotspots.map((h) => (
                    <div key={h.repo_id} className="p-4 flex justify-between items-center hover:bg-white/5">
                      <div>
                        <div className="text-xs text-white font-bold">{h.path.split("/").pop()}</div>
                        <div className="text-[10px] text-[#8b949e]">{h.repo_id}</div>
                      </div>
                      <Badge variant="destructive" className="text-[10px] font-mono">
                        {h.unresolved_failures} failures
                      </Badge>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>

          {/* Error categories */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-[10px] text-red-400 font-bold tracking-[0.15em] uppercase px-1">
              <Shield className="h-3 w-3" /> Error categories
            </div>
            <Card className="bg-[#111317] border-[#21262d]">
              <CardContent className="p-0 divide-y divide-[#21262d]">
                {errorCategories.length === 0 ? (
                  <div className="p-8 text-center text-[#8b949e] text-xs">No errors detected</div>
                ) : (
                  errorCategories.slice(0, 6).map((c) => (
                    <div key={c.category} className="p-4 flex justify-between items-center hover:bg-white/5">
                      <span className="text-xs text-white">{c.category}</span>
                      <span className="text-xs text-[#8b949e] font-mono">{c.count}</span>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>
        </div>

        {/* Live events */}
        <div className="space-y-3">
          <div className="flex items-center justify-between px-1">
            <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.15em] uppercase">Recent events</div>
            <Badge variant="outline" className="text-[9px] bg-white/5 border-[#21262d] text-white">
              {isRunning ? "LIVE" : "PAUSED"}
            </Badge>
          </div>
          <Card className="bg-[#111317] border-[#21262d] max-h-[250px] overflow-auto">
            <CardContent className="p-0 divide-y divide-[#21262d]">
              {events.length > 0 ? (
                events.slice(-15).reverse().map((event) => (
                  <div key={event.id} className="px-4 py-3 hover:bg-white/5 flex items-center gap-3">
                    <span className={cn("text-[10px] font-bold w-12",
                      event.severity === "ERROR" ? "text-red-500" : "text-primary"
                    )}>
                      {event.severity}
                    </span>
                    <span className="text-[10px] text-[#8b949e] w-20">
                      {new Date(event.timestamp * 1000).toLocaleTimeString([], { hour12: false })}
                    </span>
                    <span className="text-[11px] text-white truncate flex-1">{event.message}</span>
                  </div>
                ))
              ) : (
                <div className="p-8 text-center text-[#8b949e] text-xs">No events yet — start the watcher</div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* MCP tools */}
        <div className="space-y-3">
          <div className="text-[10px] text-primary font-bold tracking-[0.15em] uppercase px-1">
            MCP Integration — Available Tools
          </div>
          <Card className="bg-[#111317] border-[#21262d]">
            <CardContent className="p-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <h3 className="text-sm font-bold text-white mb-3">Core Tools</h3>
                  <div className="space-y-2">
                    {[
                      { name: "remember", desc: "Store memories (failures, decisions, facts)" },
                      { name: "search_memory", desc: "Semantic + keyword search" },
                      { name: "get_repo_context", desc: "Load project context at session start" },
                      { name: "global_search", desc: "Cross-repo search" },
                      { name: "list_projects", desc: "All tracked projects" },
                      { name: "switch_project_context", desc: "Full project context bundle" },
                      { name: "get_pattern_report", desc: "Health scores, trends" },
                      { name: "get_related_memories", desc: "Find related by content/time" },
                      { name: "find_similar_failures", desc: "Cross-project failure matching" },
                      { name: "failure_hotspots", desc: "Projects with most failures" },
                      { name: "mark_failure_resolved", desc: "Resolve failure signatures" },
                      { name: "forget", desc: "Soft-delete memories" },
                    ].map((tool) => (
                      <div key={tool.name} className="flex items-center gap-2">
                        <div className="h-1.5 w-1.5 rounded-full bg-green-400" />
                        <span className="text-[11px] text-primary font-bold">{tool.name}</span>
                        <span className="text-[10px] text-[#8b949e]">— {tool.desc}</span>
                      </div>
                    ))}
                  </div>
                </div>
                <div>
                  <h3 className="text-sm font-bold text-white mb-3 flex items-center gap-2">
                    <Sparkles className="h-4 w-4 text-yellow-500" /> Intelligence Tools
                    <Badge variant="outline" className="text-[8px] bg-yellow-500/10 text-yellow-400 border-yellow-500/20">NEW</Badge>
                  </h3>
                  <div className="space-y-2 mb-6">
                    {[
                      { name: "smart_context", desc: "Task-specific memory retrieval with token budget", icon: Zap },
                      { name: "memory_feedback", desc: "Report if a memory was useful (closes the loop)", icon: ThumbsUp },
                      { name: "refresh_relevance", desc: "Recompute scores + classify noise", icon: RefreshCw },
                    ].map((tool) => (
                      <div key={tool.name} className="flex items-center gap-2">
                        <div className="h-1.5 w-1.5 rounded-full bg-yellow-400" />
                        <span className="text-[11px] text-yellow-400 font-bold">{tool.name}</span>
                        <span className="text-[10px] text-[#8b949e]">— {tool.desc}</span>
                      </div>
                    ))}
                  </div>

                  <h3 className="text-sm font-bold text-white mb-3">Connection Info</h3>
                  <div className="space-y-3 text-[11px]">
                    <div className="flex justify-between">
                      <span className="text-[#8b949e]">Protocol</span>
                      <span className="text-white">MCP stdio + HTTP :9000</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-[#8b949e]">Tools</span>
                      <span className="text-green-400 font-bold">15 active</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-[#8b949e]">New endpoints</span>
                      <span className="text-yellow-400 font-bold">/api/v1/context/smart · /api/v1/feedback</span>
                    </div>
                  </div>
                  <div className="mt-4 p-3 bg-[#0a0b0d] rounded border border-[#21262d]">
                    <div className="text-[10px] text-[#8b949e] mb-2">Claude Desktop config:</div>
                    <code className="text-[10px] text-primary whitespace-pre">{`"agent-memory": {
  "command": "agent-memory",
  "args": ["mcp", "--root",
    "~/agent_companion_data"]
}`}</code>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </Layout>
  );
}

function StatCard({ icon: Icon, label, value, color, subtitle }: { icon: any; label: string; value: number | string; color: string; subtitle?: string }) {
  return (
    <Card className="bg-[#111317] border-[#21262d] hover:border-primary/20 transition-all">
      <CardContent className="p-5">
        <div className="flex items-center gap-3 mb-3">
          <div className="p-2 bg-[#1a1d23] rounded">
            <Icon className={cn("h-4 w-4", color)} />
          </div>
          <span className="text-[10px] font-bold text-[#8b949e] tracking-[0.1em] uppercase">{label}</span>
        </div>
        <p className="text-2xl font-black text-white tracking-tighter">{value}</p>
        {subtitle && <p className="text-[9px] text-[#8b949e] mt-1">{subtitle}</p>}
      </CardContent>
    </Card>
  );
}
