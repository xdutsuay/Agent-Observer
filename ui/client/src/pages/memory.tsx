import { useState, useMemo } from "react";
import { Layout } from "@/components/layout";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { 
  Database, 
  FolderGit2, 
  FileCode, 
  Tag, 
  AlertTriangle,
  ChevronRight,
  Search,
  BookOpen
} from "lucide-react";
import { useAgentSimulation } from "@/lib/simulation";
import { useQuery } from "@tanstack/react-query";
import { motion, AnimatePresence } from "framer-motion";

export default function Memory() {
  const { repos } = useAgentSimulation();
  const [selectedRepoId, setSelectedRepoId] = useState<string | null>(null);

  // Auto-select first repo if none selected
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
      <div className="p-8 max-w-7xl mx-auto">
        <header className="mb-12 flex items-center justify-between">
          <motion.div 
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            className="flex gap-6 items-end"
          >
            <div className="h-16 w-16 glass-panel p-3 animate-float ring-2 ring-primary/20">
               <Database className="h-full w-full text-primary" />
            </div>
            <div>
              <h2 className="text-3xl font-black text-white tracking-tighter uppercase italic">
                Memory_Cortex // <span className="text-primary italic">PERSISTED</span>
              </h2>
              <p className="text-muted-foreground font-mono text-xs tracking-widest uppercase">
                Accessing Decentralized Logic Buffers • Session: #8F92-A1B2
              </p>
            </div>
          </motion.div>

          <div className="flex gap-2">
             <Badge variant="outline" className="glass-panel text-[10px] py-1 px-3 text-primary border-primary/20">ENCRYPTED</Badge>
             <Badge variant="outline" className="glass-panel text-[10px] py-1 px-3 text-secondary border-secondary/20">READ_ONLY</Badge>
          </div>
        </header>

        <div className="grid grid-cols-1 lg:grid-cols-4 gap-8">
          
          {/* Sidebar: Repo Switcher */}
          <div className="space-y-6 lg:col-span-1">
             <Card className="glass-panel border-none rounded-none">
                <CardHeader className="border-b border-white/5 py-3">
                   <CardTitle className="text-[10px] font-mono text-muted-foreground uppercase tracking-[0.2em] flex items-center gap-2">
                     <FolderGit2 className="h-3 w-3" /> Workspaces
                   </CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                   <ScrollArea className="h-[400px]">
                      {Object.keys(repos).length > 0 ? Object.entries(repos).map(([id, path]) => (
                        <button
                          key={id}
                          onClick={() => setSelectedRepoId(id)}
                          className={`w-full p-4 text-left border-b border-white/5 transition-all duration-300 flex items-center justify-between group ${selectedRepoId === id ? 'bg-primary/10 border-l-2 border-l-primary' : 'hover:bg-white/5'}`}
                        >
                           <div className="truncate">
                              <div className={`text-xs font-mono font-bold mb-1 ${selectedRepoId === id ? 'text-primary' : 'text-white'}`}>
                                {path.split('/').pop()}
                              </div>
                              <div className="text-[9px] text-muted-foreground truncate opacity-60 font-mono italic">
                                ID: {id}
                              </div>
                           </div>
                           <ChevronRight className={`h-4 w-4 transition-transform group-hover:translate-x-1 ${selectedRepoId === id ? 'text-primary' : 'text-muted-foreground'}`} />
                        </button>
                      )) : (
                        <div className="p-8 text-center text-muted-foreground text-[10px] font-mono italic">
                          NO_ACTIVE_REPOS_FOUND
                        </div>
                      )}
                   </ScrollArea>
                </CardContent>
             </Card>

             <Card className="glass-panel border-none rounded-none p-4 space-y-4">
                <ContextItem icon={FileCode} label="Active Stream" value="*.py, *.log" />
                <ContextItem icon={Tag} label="Agent Version" value="OBS_V2.0.4" />
                <ContextItem icon={AlertTriangle} label="Error Count" value={memory?.signatures?.length || 0} />
             </Card>
          </div>

          {/* Main Content Area */}
          <div className="lg:col-span-3 space-y-8">
             <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                {/* Decision History */}
                <Card className="glass-panel border-none rounded-none flex flex-col h-[300px]">
                  <CardHeader className="border-b border-white/5 py-3">
                    <CardTitle className="text-[10px] font-mono text-muted-foreground uppercase tracking-[0.2em] flex items-center gap-2">
                      <BookOpen className="h-3 w-3 text-secondary" /> Decision_Ledger
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="flex-1 overflow-auto p-4 custom-scrollbar">
                     <div className="space-y-4">
                        {memory?.decisions ? (
                          <pre className="text-[11px] font-mono leading-relaxed text-white whitespace-pre-wrap">
                            {memory.decisions}
                          </pre>
                        ) : (
                          <div className="text-muted-foreground text-[10px] font-mono italic p-4 text-center">
                            AWAIT_DECISION_STREAM...
                          </div>
                        )}
                     </div>
                  </CardContent>
                </Card>

                {/* Failure Log */}
                <Card className="glass-panel border-none rounded-none flex flex-col h-[300px]">
                  <CardHeader className="border-b border-white/5 py-3">
                    <CardTitle className="text-[10px] font-mono text-muted-foreground uppercase tracking-[0.2em] flex items-center gap-2 text-accent">
                      <AlertTriangle className="h-3 w-3" /> Failure_Signatures
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="flex-1 overflow-auto p-4">
                     <div className="space-y-3">
                        {memory?.signatures?.length > 0 ? memory.signatures.map((sig: string, i: number) => (
                          <div key={i} className="text-[10px] font-mono p-3 bg-accent/5 border border-accent/20 text-accent/80 hover:bg-accent/10 transition-colors">
                             {sig}
                          </div>
                        )) : (
                          <div className="text-muted-foreground text-[10px] font-mono italic p-4 text-center">
                            INTEGRITY_COMPROMISE: 0
                          </div>
                        )}
                     </div>
                  </CardContent>
                </Card>
             </div>

             {/* Raw State Inspector */}
             <Card className="glass-panel border-none rounded-none flex flex-col min-h-[400px]">
                <CardHeader className="border-b border-white/5 flex flex-row justify-between items-center py-3">
                   <CardTitle className="text-[10px] font-mono text-muted-foreground uppercase tracking-[0.2em] flex items-center gap-2">
                      <Search className="h-3 w-3 text-primary" /> State_Inspector
                   </CardTitle>
                   <div className="flex gap-2">
                      <div className="h-1.5 w-1.5 rounded-full bg-primary animate-pulse" />
                      <div className="h-1.5 w-1.5 rounded-full bg-secondary animate-pulse [animation-delay:0.2s]" />
                      <div className="h-1.5 w-1.5 rounded-full bg-accent animate-pulse [animation-delay:0.4s]" />
                   </div>
                </CardHeader>
                <CardContent className="p-0 bg-black/40 flex-1">
                   <ScrollArea className="h-[400px] p-6">
                      <pre className="text-xs font-mono leading-loose text-blue-300/80">
                        {JSON.stringify(memory?.state || { status: "AWAITING_STATE_SYNC" }, null, 2)}
                      </pre>
                   </ScrollArea>
                </CardContent>
             </Card>
          </div>
        </div>
      </div>
    </Layout>
  );
}

function ContextItem({ icon: Icon, label, value }: any) {
  return (
    <div className="flex items-center gap-4 group">
      <div className="p-2.5 glass-panel bg-white/5 text-muted-foreground group-hover:text-primary transition-colors">
        <Icon className="h-3.5 w-3.5" />
      </div>
      <div className="min-w-0">
        <div className="text-[8px] text-muted-foreground uppercase tracking-[0.2em] mb-0.5">{label}</div>
        <div className="text-[11px] font-mono text-white truncate group-hover:text-primary transition-all">{value}</div>
      </div>
    </div>
  );
}
