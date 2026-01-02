import { Layout } from "@/components/layout";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Database, FolderGit2, FileCode, Tag } from "lucide-react";
import generatedImage from '@assets/generated_images/abstract_cybernetic_eye_representing_ai_observation.png';

export default function Memory() {
  return (
    <Layout>
      <div className="p-6 max-w-7xl mx-auto">
        <header className="mb-8 flex gap-6 items-end">
          <div className="h-24 w-24 border border-primary/50 bg-black p-1">
             <img src={generatedImage} alt="Agent Eye" className="h-full w-full object-cover opacity-80" />
          </div>
          <div>
            <h2 className="text-2xl font-bold text-white mb-2">AGENT_MEMORY_DUMP</h2>
            <p className="text-muted-foreground font-mono text-sm">
              Session ID: <span className="text-primary">8f92-a1b2-c3d4</span> • 
              Status: <span className="text-green-500">PERSISTED</span>
            </p>
          </div>
        </header>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          
          {/* Left Column: Context Tree */}
          <div className="space-y-6">
            <Card className="bg-card/50 border-border">
              <CardHeader>
                <CardTitle className="text-xs font-mono text-muted-foreground uppercase">Active Contexts</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <ContextItem 
                    icon={FolderGit2} 
                    label="Repository" 
                    value="replit/agent-watchdog" 
                    active 
                  />
                  <ContextItem 
                    icon={FileCode} 
                    label="Active File" 
                    value="src/services/observer.rs" 
                    active 
                  />
                  <ContextItem 
                    icon={Database} 
                    label="Memory Bank" 
                    value="local_v2" 
                    active 
                  />
                  <ContextItem 
                    icon={Tag} 
                    label="Task ID" 
                    value="TASK-9921" 
                  />
                </div>
              </CardContent>
            </Card>

            <Card className="bg-card/50 border-border">
              <CardHeader>
                <CardTitle className="text-xs font-mono text-muted-foreground uppercase">Decision History</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                 {[1,2,3].map((i) => (
                   <div key={i} className="text-xs font-mono border-l-2 border-border pl-3 py-1 hover:border-primary transition-colors">
                     <div className="text-muted-foreground mb-1">12:4{i}:0{i}</div>
                     <div className="text-white">Refactored error handling in main loop</div>
                   </div>
                 ))}
              </CardContent>
            </Card>
          </div>

          {/* Right Column: JSON Viewer */}
          <Card className="lg:col-span-2 bg-card/50 border-border h-full min-h-[500px] flex flex-col">
            <CardHeader className="border-b border-border flex flex-row justify-between items-center">
              <CardTitle className="text-xs font-mono text-muted-foreground uppercase">State Inspector</CardTitle>
              <div className="flex gap-2">
                 <Badge variant="outline" className="rounded-none text-[10px] border-primary text-primary">READ_ONLY</Badge>
                 <Badge variant="outline" className="rounded-none text-[10px]">JSON</Badge>
              </div>
            </CardHeader>
            <CardContent className="p-0 flex-1 bg-black/40">
              <ScrollArea className="h-[600px] p-4">
                <pre className="text-xs font-mono leading-relaxed text-blue-300">
{`{
  "agent_id": "WD-01",
  "runtime_stats": {
    "uptime_seconds": 3642,
    "events_processed": 14203,
    "last_heartbeat": "2024-03-21T14:32:01Z"
  },
  "current_task": {
    "id": "TASK-9921",
    "description": "Implement file watcher for Linux systems",
    "status": "IN_PROGRESS",
    "subtasks": [
      { "id": "ST-1", "status": "COMPLETED", "desc": "Setup inotify bindings" },
      { "id": "ST-2", "status": "PENDING", "desc": "Handle event debounce" }
    ]
  },
  "working_memory": {
    "relevant_files": [
      "src/watcher/mod.rs",
      "src/watcher/linux.rs",
      "Cargo.toml"
    ],
    "scratchpad": "Need to ensure we don't leak file descriptors when watching large trees..."
  },
  "inferred_intent": {
    "confidence": 0.94,
    "summary": "User is actively developing the core file watching logic using Rust's notify crate."
  }
}`}
                </pre>
              </ScrollArea>
            </CardContent>
          </Card>

        </div>
      </div>
    </Layout>
  );
}

function ContextItem({ icon: Icon, label, value, active }: any) {
  return (
    <div className="flex items-center gap-3 group">
      <div className={`p-2 rounded-none transition-colors ${active ? 'bg-primary/20 text-primary' : 'bg-white/5 text-muted-foreground'}`}>
        <Icon className="h-4 w-4" />
      </div>
      <div>
        <div className="text-[10px] text-muted-foreground uppercase tracking-wider mb-0.5">{label}</div>
        <div className="text-sm font-mono text-white group-hover:text-primary transition-colors">{value}</div>
      </div>
    </div>
  );
}
