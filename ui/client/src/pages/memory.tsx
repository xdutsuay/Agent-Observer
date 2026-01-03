import { useState, useMemo } from "react";
import { Layout } from "@/components/layout";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { 
  Database, 
  ChevronRight,
  Search,
  BookOpen,
  AlertTriangle
} from "lucide-react";
import { useAgentSimulation } from "@/lib/simulation";
import { useQuery } from "@tanstack/react-query";
import { cn } from "@/lib/utils";

export default function Memory() {
  const { repos } = useAgentSimulation();
  const [selectedRepoId, setSelectedRepoId] = useState<string | null>(null);

  useMemo(() => {
    if (!selectedRepoId && Object.keys(repos).length > 0) {
      setSelectedRepoId(Object.keys(repos)[0]);
    }
  }, [repos, selectedRepoId]);

  const { data: memory } = useQuery({
    queryKey: ['/api/memory', selectedRepoId],
    queryFn: async () => {
      if (!selectedRepoId) return null;
      const res = await fetch(`/api/memory/${selectedRepoId}`);
      return res.json();
    },
    enabled: !!selectedRepoId,
    refetchInterval: 5000
  });

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
                Accessing Persistent Logic Buffers
              </p>
            </div>
          </div>

          <div className="flex gap-3 text-xs font-mono">
             <span className="text-[#8b949e]">Status:</span>
             <span className="text-primary font-bold">READY</span>
          </div>
        </header>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          
          {/* Sidebar */}
          <div className="lg:col-span-3 space-y-4">
             <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.2em] uppercase px-1">
                WORKSPACES
             </div>
             <Card className="bg-[#111317] border-[#21262d] rounded-md overflow-hidden">
                <ScrollArea className="h-[500px]">
                   <div className="divide-y divide-[#21262d]">
                      {Object.keys(repos).length > 0 ? Object.entries(repos).map(([id, path]) => (
                        <button
                          key={id}
                          onClick={() => setSelectedRepoId(id)}
                          className={cn(
                            "w-full p-5 text-left transition-all flex items-center justify-between group border-l-4",
                            selectedRepoId === id ? "bg-[#161b22] border-l-primary" : "border-l-transparent hover:bg-white/5"
                          )}
                        >
                           <div className="truncate">
                              <div className={cn("text-xs font-bold mb-1 font-mono", selectedRepoId === id ? "text-primary" : "text-white")}>
                                {path.split('/').pop()}
                              </div>
                              <div className="text-[9px] text-[#8b949e] truncate opacity-50 font-mono">
                                ID: {id}
                              </div>
                           </div>
                           <ChevronRight className={cn("h-4 w-4 transition-transform group-hover:translate-x-1", selectedRepoId === id ? "text-primary" : "text-[#21262d]")} />
                        </button>
                      )) : (
                        <div className="p-10 text-center text-[#8b949e] text-[10px] font-mono italic">
                          NO_ACTIVE_REPOS_FOUND
                        </div>
                      )}
                   </div>
                </ScrollArea>
             </Card>
          </div>

          {/* Content */}
          <div className="lg:col-span-9 space-y-6">
             <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.2em] uppercase px-1 flex items-center gap-2">
                    <BookOpen className="h-3 w-3" /> DECISION_LEDGER
                  </div>
                  <Card className="bg-[#111317] border-[#21262d] rounded-md h-[400px] overflow-hidden">
                     <ScrollArea className="h-full p-6">
                        {memory?.decisions ? (
                          <pre className="text-[11px] font-mono leading-relaxed text-[#8b949e] whitespace-pre-wrap">
                            {memory.decisions}
                          </pre>
                        ) : (
                          <div className="h-full flex items-center justify-center text-[#8b949e] text-[10px] font-mono italic opacity-30">
                            AWAIT_DECISION_STREAM...
                          </div>
                        )}
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
                           {memory?.signatures?.length > 0 ? memory.signatures.map((sig: string, i: number) => (
                             <div key={i} className="text-[10px] font-mono p-4 bg-secondary/5 border border-secondary/10 text-secondary/80 rounded">
                                {sig}
                             </div>
                           )) : (
                             <div className="h-full flex items-center justify-center text-[#8b949e] text-[10px] font-mono italic opacity-30">
                               NO_FAILURES_DETECTED
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
