import { useState, useEffect } from "react";
import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Input } from "@/components/ui/input";
import {
  Database,
  ChevronRight,
  Search,
  BookOpen,
  AlertTriangle,
} from "lucide-react";
import { useAgentSimulation } from "@/lib/simulation";
import { useQuery, useMutation } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { apiRequest } from "@/lib/queryClient";

interface MemoryRecord {
  id: string;
  kind: string;
  content: string;
  source: string;
  created_at: string;
  score?: number;
}

interface FailureSig {
  signature: string;
  count: number;
  last_seen: string;
  resolved: number;
}

export default function Memory() {
  const { repos } = useAgentSimulation();
  const [selectedRepoId, setSelectedRepoId] = useState<string | null>(null);
  const [searchQ, setSearchQ] = useState("");

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

  const memories = memoriesData?.memories || [];
  const searchResults: MemoryRecord[] = searchMutation.data?.results || [];
  const displayList = searchResults.length > 0 ? searchResults : memories;
  const signatures = (failuresData?.signatures || []).filter((s) => !s.resolved);

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
                Structured records + semantic search
              </p>
            </div>
          </div>
        </header>

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
            className="px-4 bg-primary text-primary-foreground font-mono text-xs font-bold"
            onClick={() => searchQ.trim() && searchMutation.mutate(searchQ.trim())}
          >
            <Search className="h-4 w-4" />
          </button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
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

          <div className="lg:col-span-9 space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-4">
                <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.2em] uppercase px-1 flex items-center gap-2">
                  <BookOpen className="h-3 w-3" /> MEMORY_RECORDS
                </div>
                <Card className="bg-[#111317] border-[#21262d] rounded-md h-[400px] overflow-hidden">
                  <ScrollArea className="h-full p-4">
                    <div className="space-y-3">
                      {displayList.map((m) => (
                        <div
                          key={m.id}
                          className="p-3 border border-[#21262d] rounded font-mono text-[10px]"
                        >
                          <div className="flex justify-between mb-2">
                            <Badge variant="outline" className="text-[8px]">
                              {m.kind}
                            </Badge>
                            <span className="text-[#8b949e]">{m.source}</span>
                          </div>
                          <p className="text-[#8b949e] whitespace-pre-wrap line-clamp-4">
                            {m.content}
                          </p>
                          <div className="text-[8px] text-[#8b949e] mt-2 opacity-60">
                            {m.created_at}
                            {m.score != null && ` · score ${m.score.toFixed(2)}`}
                          </div>
                        </div>
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

              <div className="space-y-4">
                <div className="text-[10px] text-secondary font-bold tracking-[0.2em] uppercase px-1 flex items-center gap-2">
                  <AlertTriangle className="h-3 w-3" /> FAILURE_SIGNATURES
                </div>
                <Card className="bg-[#111317] border-[#21262d] rounded-md h-[400px] overflow-hidden">
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
