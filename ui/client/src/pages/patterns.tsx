import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useQuery } from "@tanstack/react-query";
import { BarChart3, AlertTriangle, TrendingUp, Shield } from "lucide-react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from "recharts";
import { cn } from "@/lib/utils";

interface PatternReport {
  health_score: { score: number; grade: string; reasons: string[] };
  activity_trends: { total: number; by_day: Record<string, number>; by_kind: Record<string, number> };
  common_error_categories: { category: string; count: number }[];
  recurring_failures: { signature: string; count: number; first_seen?: string; last_seen?: string; resolved?: number; repos?: string[] }[];
  decision_patterns: { total_decisions: number; top_topics: { topic: string; count: number }[] };
}

const COLORS = ["#00e5ff", "#f97316", "#ef4444", "#a855f7", "#22c55e", "#eab308", "#06b6d4", "#ec4899"];

export default function Patterns() {
  const { data: patterns } = useQuery<PatternReport>({
    queryKey: ["/api/patterns"],
    refetchInterval: 10000,
  });

  if (!patterns) {
    return (
      <Layout>
        <div className="p-8 flex items-center justify-center h-full">
          <div className="text-[#8b949e] text-xs">Loading pattern analysis…</div>
        </div>
      </Layout>
    );
  }

  const health = patterns.health_score;
  const byDay = Object.entries(patterns.activity_trends.by_day || {})
    .map(([day, count]) => ({ day: day.slice(5), count }))
    .slice(-14);
  const byKind = Object.entries(patterns.activity_trends.by_kind || {})
    .map(([kind, count]) => ({ kind, count }));
  const errorCats = patterns.common_error_categories || [];
  const recurring = patterns.recurring_failures || [];

  const gradeColor = (g: string) => {
    if (g === "A") return "text-green-400 border-green-400/30";
    if (g === "B") return "text-blue-400 border-blue-400/30";
    if (g === "C") return "text-yellow-400 border-yellow-400/30";
    if (g === "D") return "text-orange-400 border-orange-400/30";
    return "text-red-400 border-red-400/30";
  };

  return (
    <Layout>
      <div className="p-8 space-y-8">
        <header className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-3 mb-1">
              <BarChart3 className="h-6 w-6 text-primary" />
              <h1 className="text-2xl font-black text-white uppercase">Pattern Analysis</h1>
            </div>
            <p className="text-xs text-[#8b949e]">
              {patterns.activity_trends.total} total memories analyzed
            </p>
          </div>

          {/* Big health score */}
          <div className={cn("border-2 rounded-xl p-4 text-center min-w-[100px]", gradeColor(health.grade))}>
            <div className="text-4xl font-black">{health.grade}</div>
            <div className="text-xs opacity-70">{health.score}/100</div>
          </div>
        </header>

        {/* Health reasons */}
        {health.reasons.length > 0 && (
          <Card className="bg-[#111317] border-[#21262d]">
            <CardContent className="p-4 space-y-1">
              {health.reasons.map((r, i) => (
                <div key={i} className="text-[11px] text-[#8b949e] flex items-center gap-2">
                  <span className="text-orange-400">•</span> {r}
                </div>
              ))}
            </CardContent>
          </Card>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Activity by day */}
          <div className="lg:col-span-8 space-y-3">
            <div className="flex items-center gap-2 text-[10px] text-[#8b949e] font-bold tracking-[0.15em] uppercase px-1">
              <TrendingUp className="h-3 w-3" /> Activity (last 14 days)
            </div>
            <Card className="bg-[#111317] border-[#21262d] h-[280px]">
              <div className="h-full w-full p-4">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={byDay} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
                    <XAxis dataKey="day" tick={{ fill: "#8b949e", fontSize: 9 }} />
                    <YAxis tick={{ fill: "#8b949e", fontSize: 9 }} />
                    <Tooltip contentStyle={{ background: "#111317", border: "1px solid #21262d", fontSize: 11 }} />
                    <Bar dataKey="count" fill="#00e5ff" radius={[3, 3, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          </div>

          {/* Memory types pie */}
          <div className="lg:col-span-4 space-y-3">
            <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.15em] uppercase px-1">
              Memory types
            </div>
            <Card className="bg-[#111317] border-[#21262d] h-[280px]">
              <div className="h-full w-full p-4">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie data={byKind} dataKey="count" nameKey="kind" cx="50%" cy="50%" outerRadius={80} strokeWidth={0}>
                      {byKind.map((_, i) => (
                        <Cell key={i} fill={COLORS[i % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip contentStyle={{ background: "#111317", border: "1px solid #21262d", fontSize: 11 }} />
                  </PieChart>
                </ResponsiveContainer>
                <div className="flex flex-wrap gap-2 justify-center">
                  {byKind.map((k, i) => (
                    <div key={k.kind} className="flex items-center gap-1 text-[9px] text-[#8b949e]">
                      <div className="h-2 w-2 rounded-full" style={{ background: COLORS[i % COLORS.length] }} />
                      {k.kind}
                    </div>
                  ))}
                </div>
              </div>
            </Card>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Recurring failures */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-[10px] text-orange-400 font-bold tracking-[0.15em] uppercase px-1">
              <AlertTriangle className="h-3 w-3" /> Recurring failures ({recurring.length})
            </div>
            <Card className="bg-[#111317] border-[#21262d] max-h-[350px] overflow-auto">
              <CardContent className="p-0 divide-y divide-[#21262d]">
                {recurring.length === 0 ? (
                  <div className="p-8 text-center text-[#8b949e] text-xs">No recurring failures</div>
                ) : (
                  recurring.map((f) => (
                    <div key={f.signature} className="p-4 hover:bg-white/5">
                      <div className="text-[11px] text-white font-mono truncate mb-1">{f.signature}</div>
                      <div className="flex items-center gap-3">
                        <Badge variant="destructive" className="text-[9px]">{f.count}x</Badge>
                        {f.resolved != null && (
                          <span className="text-[9px] text-[#8b949e]">
                            {f.resolved ? "resolved" : "unresolved"}
                          </span>
                        )}
                        {f.repos && (
                          <span className="text-[9px] text-[#8b949e]">
                            in {f.repos.join(", ")}
                          </span>
                        )}
                      </div>
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
            <Card className="bg-[#111317] border-[#21262d] max-h-[350px] overflow-auto">
              <CardContent className="p-0 divide-y divide-[#21262d]">
                {errorCats.length === 0 ? (
                  <div className="p-8 text-center text-[#8b949e] text-xs">No errors</div>
                ) : (
                  errorCats.map((c) => (
                    <div key={c.category} className="p-4 flex justify-between items-center hover:bg-white/5">
                      <span className="text-[11px] text-white">{c.category}</span>
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 bg-red-500/30 rounded-full overflow-hidden w-20">
                          <div
                            className="h-full bg-red-500 rounded-full"
                            style={{ width: `${Math.min(100, (c.count / (errorCats[0]?.count || 1)) * 100)}%` }}
                          />
                        </div>
                        <span className="text-[10px] text-[#8b949e] font-mono w-8 text-right">{c.count}</span>
                      </div>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </Layout>
  );
}
