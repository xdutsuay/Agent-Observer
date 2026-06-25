import { useState, useEffect } from "react";
import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Database,
  ChevronRight,
  Search,
  BookOpen,
  AlertTriangle,
  ThumbsUp,
  ThumbsDown,
  Sparkles,
  RefreshCw,
  Trash2,
  Eye,
} from "lucide-react";
import { useAgentSimulation } from "@/lib/simulation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { apiRequest } from "@/lib/queryClient";

interface MemoryRecord {
  id: string;
  kind: string;
  content: string;
  source: string;
  created_at: string;
  score?: number;
  relevance_score?: number;
  quality_tier?: string;
  access_count?: number;
  last_accessed?: string;
}

interface FailureSig {
  signature: string;
  count: number;
  last_seen: string;
  resolved: number;
}

const tierColor = (tier?: string) => {
  switch (tier) {
    case "high": return "bg-green-500/10 text-green-400 border-green-500/20";
    case "medium": return "bg-blue-500/10 text-blue-400 border-blue-500/20";
    case "low": return "bg-yellow-500/10 text-yellow-400 border-yellow-500/20";
    case "noise": return "bg-red-500/10 text-red-400 border-red-500/20";
    default: return "bg-white/5 text-[#8b949e] border-[#21262d]";
  }
};

const kindColor = (kind: string) => {
  switch (kind) {
    case "failure": return "text-red-400 border-red-400/30";
    case "decision": return "text-blue-400 border-blue-400/30";
    case "fact": return "text-green-400 border-green-400/30";
    case "preference": return "text-purple-400 border-purple-400/30";
    default: return "text-[#8b949e] border-[#21262d]";
  }
};

const relevanceBar = (score: number) => {
  const pct = Math.round(score * 100);
  const color =
    pct >= 60 ? "bg-green-500" :
    pct >= 30 ? "bg-yellow-500" :
    "bg-red-500/60";
  return (
    <div className="flex items-center gap-2">
      <div className="w-16 h-1.5 bg-[#21262d] rounded-full overflow-hidden">
        <div className={cn("h-full rounded-full transition-all", color)} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-[9px] text-[#8b949e] font-mono">{pct}%</span>
    </div>
  );
};

export default function Memory() {
  const { repos } = useAgentSimulation();
  const [selectedRepoId, setSelectedRepoId] = useState<string | null>(null);
  const [searchQ, setSearchQ] = useState("");
  const qc = useQueryClient();

  useEffect(() => {
    if (!selectedRepoId && repos.length > 0) {
      setSelectedRepoId(repos[0].id);
    }
  }, [repos, selectedRepoId]);

  const { data: memoriesData } = useQuery<{ memories: MemoryRecord[] }>({
    queryKey: ["/api/memories", selectedRepoId],
    queryFn: async () => {
      const res = await fetch(`/api/memories?repo_id=${selectedRepoId}&limit=40`);
      return res.json();
    },
    enabled: !!selectedRepoId,
    refetchInterval: 5000,
  });

  const { data: failuresData } = useQuery<{ signatures: FailureSig[] }>({
    queryKey: ["/api/failures", selectedRepoId],
    queryFn: async () => {
      const res = await fetch(`/api/failures/${selectedRepoId}`);
      return res.json();
    },
    enabled: !!selectedRepoId,
    refetchInterval: 5000,
  });

  const searchMutation = useMutation({
    mutationFn: async (query: string) => {
      const res = await apiRequest("POST", "/api/search", {
        query,
        repo_id: selectedRepoId,
        limit: 20,
      });
      return res.json();
    },
  });

  const feedbackMutation = useMutation({
    mutationFn: async ({ memoryId, useful }: { memoryId: string; useful: boolean }) => {
      const res = await apiRequest("POST", "/api/v1/feedback", {
        memory_id: memoryId,
        useful,
      });
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["/api/memories", selectedRepoId] });
    },
  });

  const refreshMutation = useMutation({
    mutationFn: async () => {
      const res = await apiRequest("POST", `/api/v1/relevance/refresh?repo_id=${selectedRepoId}`);
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["/api/memories", selectedRepoId] });
    },
  });

  const memories = memoriesData?.memories || [];
  const searchResults: MemoryRecord[] = searchMutation.data?.results || [];
  const displayList = searchResults.length > 0 ? searchResults : memories;
  const signatures = (failuresData?.signatures || []).filter((s) => !s.resolved);

  // Stats
  const noiseCount = memories.filter((m) => m.quality_tier === "noise").length;
  const avgRelevance = memories.length > 0
    ? memories.reduce((s, m) => s + (m.relevance_score || 0), 0) / memories.length
    : 0;
  const totalAccesses = memories.reduce((s, m) => s + (m.access_count || 0), 0);

  return (
    <Layout>
      <div className="p-10 max-w-full mx-auto space-y-8">
        <header className="flex items-center justify-between">
          <div className="flex gap-4 items-center">
            <div className="h-12 w-12 bg-primary/10 flex items-center justify-center rounded">
              <Database className="h-6 w-6 text-primary" />
            </div>
            <div>
              <h2 className="text-2xl font-black text-white tracking-tight uppercase font-mono">
                AGENT MEMORY
              </h2>
              <p className="text-[#8b949e] font-mono text-[10px] tracking-widest uppercase">
                Structured records + relevance scoring + semantic search
              </p>
            </div>
          </div>
          <Button
            variant="outline"
            size="sm"
            className="text-[10px] font-mono gap-1"
            onClick={() => refreshMutation.mutate()}
            disabled={refreshMutation.isPending}
          >
            <RefreshCw className={cn("h-3 w-3", refreshMutation.isPending && "animate-spin")} />
            {refreshMutation.isPending ? "Refreshing..." : "Refresh Relevance"}
          </Button>
        </header>

        {/* Relevance stats bar */}
        <div className="grid grid-cols-4 gap-3">
          <MiniStat label="MEMORIES" value={memories.length} />
          <MiniStat label="AVG RELEVANCE" value={`${Math.round(avgRelevance * 100)}%`} />
          <MiniStat label="TOTAL ACCESSES" value={totalAccesses} />
          <MiniStat label="NOISE" value={noiseCount} accent={noiseCount > 0 ? "text-red-400" : undefined} />
        </div>

        <div className="flex gap-2 max-w-xl">
          <Input
            className="font-mono text-xs bg-[#111317] border-[#21262d]"
            placeholder="Search memories..."
            value={searchQ}
            onChange={(e) => setSearchQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && searchQ.trim()) {
                searchMutation.mutate(searchQ.trim());
              }
            }}
          />
          <button
            className="px-4 bg-primary text-primary-foreground font-mono text-xs font-bold rounded"
            onClick={() => searchQ.trim() && searchMutation.mutate(searchQ.trim())}
          >
            <Search className="h-4 w-4" />
          </button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Sidebar: workspaces */}
          <div className="lg:col-span-3 space-y-4">
            <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.2em] uppercase px-1">
              WORKSPACES
            </div>
            <Card className="bg-[#111317] border-[#21262d] rounded-md overflow-hidden">
              <ScrollArea className="h-[500px]">
                <div className="divide-y divide-[#21262d]">
                  {repos.map((repo) => (
                    <button
                      key={repo.id}
                      onClick={() => {
                        setSelectedRepoId(repo.id);
                        searchMutation.reset();
                      }}
                      className={cn(
                        "w-full p-5 text-left transition-all flex items-center justify-between group border-l-4",
                        selectedRepoId === repo.id
                          ? "bg-[#161b22] border-l-primary"
                          : "border-l-transparent hover:bg-white/5"
                      )}
                    >
                      <div className="truncate pr-2">
                        <div
                          className={cn(
                            "text-xs font-bold mb-1 font-mono",
                            selectedRepoId === repo.id ? "text-primary" : "text-white"
                          )}
                        >
                          {repo.path.split("/").pop()}
                        </div>
                        <div className="text-[9px] text-[#8b949e] truncate opacity-50 font-mono">
                          {repo.id}
                        </div>
                      </div>
                      {repo.error_count > 0 && (
                        <Badge variant="destructive" className="text-[8px] py-0 px-1 font-mono">
                          {repo.error_count} ERR
                        </Badge>
                      )}
                      <ChevronRight className="h-4 w-4" />
                    </button>
                  ))}
                </div>
              </ScrollArea>
            </Card>
          </div>

          {/* Main: memory records + failures */}
          <div className="lg:col-span-9 space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Memory records */}
              <div className="space-y-4">
                <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.2em] uppercase px-1 flex items-center gap-2">
                  <BookOpen className="h-3 w-3" /> MEMORY_RECORDS
                  {searchResults.length > 0 && (
                    <Badge variant="outline" className="text-[8px] ml-2">
                      {searchResults.length} search results
                    </Badge>
                  )}
                </div>
                <Card className="bg-[#111317] border-[#21262d] rounded-md h-[500px] overflow-hidden">
                  <ScrollArea className="h-full p-4">
                    <div className="space-y-3">
                      {displayList.map((m) => (
                        <MemoryCard
                          key={m.id}
                          memory={m}
                          onFeedback={(useful) =>
                            feedbackMutation.mutate({ memoryId: m.id, useful })
                          }
                        />
                      ))}
                      {displayList.length === 0 && (
                        <div className="text-center text-[#8b949e] italic py-20">
                          NO_RECORDS
                        </div>
                      )}
                    </div>
                  </ScrollArea>
                </Card>
              </div>

              {/* Failure signatures */}
              <div className="space-y-4">
                <div className="text-[10px] text-secondary font-bold tracking-[0.2em] uppercase px-1 flex items-center gap-2">
                  <AlertTriangle className="h-3 w-3" /> FAILURE_SIGNATURES
                </div>
                <Card className="bg-[#111317] border-[#21262d] rounded-md h-[500px] overflow-hidden">
                  <ScrollArea className="h-full p-6">
                    <div className="space-y-3">
                      {signatures.map((data, i) => (
                        <div
                          key={i}
                          className="text-[10px] font-mono p-4 bg-secondary/5 border border-secondary/10 text-secondary/80 rounded flex justify-between"
                        >
                          <div className="truncate pr-4">{data.signature}</div>
                          <Badge variant="outline" className="text-[9px]">
                            {data.count}x
                          </Badge>
                        </div>
                      ))}
                      {signatures.length === 0 && (
                        <div className="text-center text-[#8b949e] italic py-20 opacity-30">
                          NO_UNRESOLVED_FAILURES
                        </div>
                      )}
                    </div>
                  </ScrollArea>
                </Card>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}

function MemoryCard({
  memory: m,
  onFeedback,
}: {
  memory: MemoryRecord;
  onFeedback: (useful: boolean) => void;
}) {
  return (
    <div className="p-3 border border-[#21262d] rounded font-mono text-[10px] group hover:border-primary/20 transition-all">
      {/* Header: kind + tier + source */}
      <div className="flex justify-between items-center mb-2">
        <div className="flex items-center gap-1.5">
          <Badge variant="outline" className={cn("text-[8px]", kindColor(m.kind))}>
            {m.kind}
          </Badge>
          {m.quality_tier && m.quality_tier !== "unrated" && (
            <Badge variant="outline" className={cn("text-[7px] px-1", tierColor(m.quality_tier))}>
              {m.quality_tier}
            </Badge>
          )}
        </div>
        <span className="text-[#8b949e]">{m.source}</span>
      </div>

      {/* Content */}
      <p className="text-[#8b949e] whitespace-pre-wrap line-clamp-4">{m.content}</p>

      {/* Footer: relevance + access + feedback */}
      <div className="flex items-center justify-between mt-2 pt-2 border-t border-[#21262d]/50">
        <div className="flex items-center gap-3">
          {/* Relevance bar */}
          {m.relevance_score != null && m.relevance_score > 0 && (
            <div className="flex items-center gap-1">
              <Sparkles className="h-2.5 w-2.5 text-yellow-500" />
              {relevanceBar(m.relevance_score)}
            </div>
          )}

          {/* Access count */}
          {(m.access_count ?? 0) > 0 && (
            <div className="flex items-center gap-1 text-[9px] text-[#8b949e]">
              <Eye className="h-2.5 w-2.5" />
              {m.access_count}
            </div>
          )}

          {/* Search score */}
          {m.score != null && (
            <span className="text-[9px] text-[#8b949e]">
              match: {(m.score * 100).toFixed(0)}%
            </span>
          )}
        </div>

        {/* Feedback buttons */}
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            className="p-1 hover:bg-green-500/10 rounded transition-colors"
            onClick={() => onFeedback(true)}
            title="Mark as useful"
          >
            <ThumbsUp className="h-3 w-3 text-green-500" />
          </button>
          <button
            className="p-1 hover:bg-red-500/10 rounded transition-colors"
            onClick={() => onFeedback(false)}
            title="Mark as not useful"
          >
            <ThumbsDown className="h-3 w-3 text-red-500" />
          </button>
        </div>
      </div>

      {/* Timestamp */}
      <div className="text-[8px] text-[#8b949e] mt-1 opacity-60">{m.created_at}</div>
    </div>
  );
}

function MiniStat({ label, value, accent }: { label: string; value: string | number; accent?: string }) {
  return (
    <div className="bg-[#111317] border border-[#21262d] rounded p-3">
      <div className="text-[8px] text-[#8b949e] uppercase tracking-widest mb-1">{label}</div>
      <div className={cn("text-lg font-black", accent || "text-white")}>{value}</div>
    </div>
  );
}
