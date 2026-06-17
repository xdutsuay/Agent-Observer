import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Search, Clock, FolderOpen } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";

interface SearchResult {
  id: string;
  repo_id: string;
  kind: string;
  content: string;
  created_at: number;
  score?: number;
}

export default function SearchPage() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);

  const doSearch = async () => {
    if (!query.trim()) return;
    setLoading(true);
    setSearched(true);
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

  const kindColor = (kind: string) => {
    if (kind === "failure") return "text-red-400 border-red-400/30 bg-red-400/5";
    if (kind === "decision") return "text-blue-400 border-blue-400/30 bg-blue-400/5";
    if (kind === "fact") return "text-green-400 border-green-400/30 bg-green-400/5";
    return "text-[#8b949e] border-[#21262d] bg-white/5";
  };

  return (
    <Layout>
      <div className="p-8 space-y-6">
        <header>
          <div className="flex items-center gap-3 mb-1">
            <Search className="h-6 w-6 text-primary" />
            <h1 className="text-2xl font-black text-white uppercase">Cross-Repo Search</h1>
          </div>
          <p className="text-xs text-[#8b949e]">
            Search across all projects — semantic + keyword matching
          </p>
        </header>

        {/* Search bar */}
        <div className="flex gap-3">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && doSearch()}
            placeholder="Search memories across all repos…"
            className="bg-[#111317] border-[#21262d] text-white placeholder:text-[#8b949e] flex-1"
          />
          <Button onClick={doSearch} disabled={loading} className="bg-primary text-black font-bold hover:bg-primary/80">
            {loading ? "Searching…" : "Search"}
          </Button>
        </div>

        {/* Results */}
        {searched && (
          <div className="space-y-3">
            <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.15em] uppercase px-1">
              {results.length} results
            </div>

            {results.length === 0 ? (
              <Card className="bg-[#111317] border-[#21262d]">
                <CardContent className="p-12 text-center text-[#8b949e] text-xs">
                  No matches found for "{query}"
                </CardContent>
              </Card>
            ) : (
              <div className="space-y-2">
                {results.map((r) => (
                  <Card key={r.id} className="bg-[#111317] border-[#21262d] hover:border-primary/20 transition-all">
                    <CardContent className="p-4">
                      <div className="flex items-start justify-between gap-4 mb-2">
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className={cn("text-[9px]", kindColor(r.kind))}>
                            {r.kind}
                          </Badge>
                          <div className="flex items-center gap-1 text-[10px] text-[#8b949e]">
                            <FolderOpen className="h-3 w-3" />
                            {r.repo_id}
                          </div>
                        </div>
                        <div className="flex items-center gap-1 text-[10px] text-[#8b949e] shrink-0">
                          <Clock className="h-3 w-3" />
                          {new Date(r.created_at * 1000).toLocaleDateString()}
                        </div>
                      </div>
                      <div className="text-[11px] text-white leading-relaxed">
                        {r.content.length > 300 ? r.content.slice(0, 300) + "…" : r.content}
                      </div>
                      {r.score != null && (
                        <div className="mt-2 text-[9px] text-[#8b949e]">
                          relevance: {(r.score * 100).toFixed(0)}%
                        </div>
                      )}
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
