import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Search,
  Clock,
  FolderOpen,
  Sparkles,
  Eye,
  ThumbsUp,
  ThumbsDown,
  Zap,
  Copy,
} from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import { useMutation } from "@tanstack/react-query";
import { apiRequest } from "@/lib/queryClient";

interface SearchResult {
  id: string;
  repo_id: string;
  kind: string;
  content: string;
  created_at: string;
  score?: number;
  relevance_score?: number;
  quality_tier?: string;
  access_count?: number;
}

interface SmartContextResponse {
  memories: SearchResult[];
  token_estimate: number;
  system_prompt_fragment: string;
}

const kindColor = (kind: string) => {
  if (kind === "failure") return "text-red-400 border-red-400/30 bg-red-400/5";
  if (kind === "decision") return "text-blue-400 border-blue-400/30 bg-blue-400/5";
  if (kind === "fact") return "text-green-400 border-green-400/30 bg-green-400/5";
  if (kind === "preference") return "text-purple-400 border-purple-400/30 bg-purple-400/5";
  return "text-[#8b949e] border-[#21262d] bg-white/5";
};

const tierBadge = (tier?: string) => {
  if (!tier || tier === "unrated") return null;
  const colors: Record<string, string> = {
    high: "bg-green-500/10 text-green-400 border-green-500/20",
    medium: "bg-blue-500/10 text-blue-400 border-blue-500/20",
    low: "bg-yellow-500/10 text-yellow-400 border-yellow-500/20",
    noise: "bg-red-500/10 text-red-400 border-red-500/20",
  };
  return (
    <Badge variant="outline" className={cn("text-[7px] px-1", colors[tier] || "")}>
      {tier}
    </Badge>
  );
};

export default function SearchPage() {
  const [query, setQuery] = useState("");
  const [repoFilter, setRepoFilter] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [smartContext, setSmartContext] = useState<SmartContextResponse | null>(null);
  const [copied, setCopied] = useState(false);

  const doSearch = async () => {
    if (!query.trim()) return;
    setLoading(true);
    setSearched(true);
    setSmartContext(null);
    try {
      const res = await fetch("/api/search/global", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: query.trim(), limit: 50 }),
      });
      const data = await res.json();
      setResults(data.results || []);
    } catch {
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const doSmartContext = async () => {
    if (!query.trim() || !repoFilter.trim()) return;
    setLoading(true);
    setSearched(true);
    setResults([]);
    try {
      const res = await fetch("/api/v1/context/smart", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          task: query.trim(),
          repo_id: repoFilter.trim(),
          max_tokens: 3000,
        }),
      });
      const data: SmartContextResponse = await res.json();
      setSmartContext(data);
    } catch {
      setSmartContext(null);
    } finally {
      setLoading(false);
    }
  };

  const feedbackMutation = useMutation({
    mutationFn: async ({ memoryId, useful }: { memoryId: string; useful: boolean }) => {
      const res = await apiRequest("POST", "/api/v1/feedback", {
        memory_id: memoryId,
        useful,
      });
      return res.json();
    },
  });

  const copyPrompt = () => {
    if (smartContext?.system_prompt_fragment) {
      navigator.clipboard.writeText(smartContext.system_prompt_fragment);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const displayResults = smartContext?.memories || results;

  return (
    <Layout>
      <div className="p-8 space-y-6">
        <header>
          <div className="flex items-center gap-3 mb-1">
            <Search className="h-6 w-6 text-primary" />
            <h1 className="text-2xl font-black text-white uppercase">Cross-Repo Search</h1>
          </div>
          <p className="text-xs text-[#8b949e]">
            Search across all projects — semantic + keyword + relevance scoring
          </p>
        </header>

        <Tabs defaultValue="search" className="space-y-4">
          <TabsList className="bg-[#111317] border border-[#21262d]">
            <TabsTrigger value="search" className="text-xs gap-1">
              <Search className="h-3 w-3" /> Global Search
            </TabsTrigger>
            <TabsTrigger value="smart" className="text-xs gap-1">
              <Zap className="h-3 w-3" /> Smart Context
            </TabsTrigger>
          </TabsList>

          <TabsContent value="search" className="space-y-4">
            <div className="flex gap-3">
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && doSearch()}
                placeholder="Search memories across all repos..."
                className="bg-[#111317] border-[#21262d] text-white placeholder:text-[#8b949e] flex-1"
              />
              <Button onClick={doSearch} disabled={loading} className="bg-primary text-black font-bold hover:bg-primary/80">
                {loading ? "Searching..." : "Search"}
              </Button>
            </div>
          </TabsContent>

          <TabsContent value="smart" className="space-y-4">
            <div className="flex gap-3">
              <Input
                value={repoFilter}
                onChange={(e) => setRepoFilter(e.target.value)}
                placeholder="repo_id (e.g. abc123)"
                className="bg-[#111317] border-[#21262d] text-white placeholder:text-[#8b949e] w-48"
              />
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && doSmartContext()}
                placeholder="Describe the task you're about to work on..."
                className="bg-[#111317] border-[#21262d] text-white placeholder:text-[#8b949e] flex-1"
              />
              <Button onClick={doSmartContext} disabled={loading || !repoFilter.trim()} className="bg-primary text-black font-bold hover:bg-primary/80 gap-1">
                <Zap className="h-3 w-3" />
                {loading ? "Loading..." : "Get Context"}
              </Button>
            </div>
            <p className="text-[10px] text-[#8b949e]">
              Returns the most relevant memories for your task, ranked by relevance and packed into a token budget.
            </p>
          </TabsContent>
        </Tabs>

        {/* Smart context prompt fragment */}
        {smartContext && smartContext.system_prompt_fragment && (
          <Card className="bg-[#0a0b0d] border-primary/20">
            <CardContent className="p-4">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <Zap className="h-4 w-4 text-primary" />
                  <span className="text-[10px] text-primary font-bold uppercase tracking-widest">
                    Generated System Prompt Fragment
                  </span>
                  <Badge variant="outline" className="text-[8px]">
                    ~{smartContext.token_estimate} tokens
                  </Badge>
                </div>
                <Button variant="ghost" size="sm" className="text-[10px] gap-1 h-6" onClick={copyPrompt}>
                  <Copy className="h-3 w-3" />
                  {copied ? "Copied!" : "Copy"}
                </Button>
              </div>
              <pre className="text-[10px] text-[#8b949e] font-mono whitespace-pre-wrap max-h-40 overflow-auto">
                {smartContext.system_prompt_fragment}
              </pre>
            </CardContent>
          </Card>
        )}

        {/* Results */}
        {searched && (
          <div className="space-y-3">
            <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.15em] uppercase px-1">
              {displayResults.length} results
              {smartContext && ` · ~${smartContext.token_estimate} tokens`}
            </div>

            {displayResults.length === 0 ? (
              <Card className="bg-[#111317] border-[#21262d]">
                <CardContent className="p-12 text-center text-[#8b949e] text-xs">
                  No matches found for "{query}"
                </CardContent>
              </Card>
            ) : (
              <div className="space-y-2">
                {displayResults.map((r) => (
                  <Card key={r.id} className="bg-[#111317] border-[#21262d] hover:border-primary/20 transition-all group">
                    <CardContent className="p-4">
                      <div className="flex items-start justify-between gap-4 mb-2">
                        <div className="flex items-center gap-2 flex-wrap">
                          <Badge variant="outline" className={cn("text-[9px]", kindColor(r.kind))}>
                            {r.kind}
                          </Badge>
                          {tierBadge(r.quality_tier)}
                          <div className="flex items-center gap-1 text-[10px] text-[#8b949e]">
                            <FolderOpen className="h-3 w-3" />
                            {r.repo_id}
                          </div>
                        </div>
                        <div className="flex items-center gap-3 shrink-0">
                          {/* Feedback */}
                          <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button
                              className="p-1 hover:bg-green-500/10 rounded"
                              onClick={() => feedbackMutation.mutate({ memoryId: r.id, useful: true })}
                              title="Useful"
                            >
                              <ThumbsUp className="h-3 w-3 text-green-500" />
                            </button>
                            <button
                              className="p-1 hover:bg-red-500/10 rounded"
                              onClick={() => feedbackMutation.mutate({ memoryId: r.id, useful: false })}
                              title="Not useful"
                            >
                              <ThumbsDown className="h-3 w-3 text-red-500" />
                            </button>
                          </div>
                          <div className="flex items-center gap-1 text-[10px] text-[#8b949e]">
                            <Clock className="h-3 w-3" />
                            {r.created_at?.slice(0, 10)}
                          </div>
                        </div>
                      </div>

                      <div className="text-[11px] text-white leading-relaxed">
                        {r.content.length > 300 ? r.content.slice(0, 300) + "..." : r.content}
                      </div>

                      {/* Score bar */}
                      <div className="flex items-center gap-4 mt-2 pt-2 border-t border-[#21262d]/50">
                        {r.score != null && (
                          <div className="flex items-center gap-1">
                            <span className="text-[8px] text-[#8b949e] uppercase">match</span>
                            <ScoreBar value={r.score} />
                          </div>
                        )}
                        {r.relevance_score != null && r.relevance_score > 0 && (
                          <div className="flex items-center gap-1">
                            <Sparkles className="h-2.5 w-2.5 text-yellow-500" />
                            <span className="text-[8px] text-[#8b949e] uppercase">relevance</span>
                            <ScoreBar value={r.relevance_score} />
                          </div>
                        )}
                        {(r.access_count ?? 0) > 0 && (
                          <div className="flex items-center gap-1 text-[9px] text-[#8b949e]">
                            <Eye className="h-2.5 w-2.5" />
                            {r.access_count} views
                          </div>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </Layout>
  );
}

function ScoreBar({ value }: { value: number }) {
  const pct = Math.round(Math.min(value, 1) * 100);
  const color = pct >= 60 ? "bg-green-500" : pct >= 30 ? "bg-yellow-500" : "bg-red-500/60";
  return (
    <div className="flex items-center gap-1.5">
      <div className="w-12 h-1.5 bg-[#21262d] rounded-full overflow-hidden">
        <div className={cn("h-full rounded-full", color)} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-[8px] text-[#8b949e] font-mono w-6">{pct}%</span>
    </div>
  );
}
