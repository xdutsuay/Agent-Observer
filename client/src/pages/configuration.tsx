import { Layout } from "@/components/layout";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Save } from "lucide-react";

export default function Configuration() {
  const defaultConfig = `# Watchdog Service Configuration
# Version: 2.1.0

service:
  id: "wd-node-01"
  mode: "active_observation"
  log_level: "debug"
  
heuristics:
  process_scanning:
    enabled: true
    interval_ms: 500
    target_processes:
      - "python.exe"
      - "node"
      - "cargo"
  
  filesystem:
    watch_paths:
      - "/home/user/workspace"
      - "/tmp/agent-memory"
    ignore_patterns:
      - "**/node_modules/**"
      - "**/.git/**"
      
storage:
  type: "local_json"
  path: "./data/memory.json"
  rotate_logs: true
  max_size_mb: 500

cloud_routing:
  enabled: false
  endpoint: "wss://api.antigravity.dev/v1/stream"
`;

  return (
    <Layout>
      <div className="p-6 max-w-7xl mx-auto h-full flex flex-col">
         <header className="flex justify-between items-center mb-8">
            <div>
              <h2 className="text-2xl font-bold text-white mb-2">CONFIGURATION</h2>
              <p className="text-muted-foreground">Manage service parameters and heuristic thresholds.</p>
            </div>
            <Button className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-none font-mono">
              <Save className="mr-2 h-4 w-4" />
              APPLY_CHANGES
            </Button>
          </header>

          <Card className="flex-1 bg-card/50 border-border overflow-hidden flex flex-col">
            <CardHeader className="border-b border-border bg-black/20">
              <CardTitle className="text-xs font-mono text-muted-foreground flex gap-4">
                <span className="text-primary border-b border-primary pb-4 -mb-4 px-2">config.yaml</span>
                <span className="hover:text-white cursor-pointer px-2 transition-colors">heuristics.json</span>
                <span className="hover:text-white cursor-pointer px-2 transition-colors">routing_table.csv</span>
              </CardTitle>
            </CardHeader>
            <CardContent className="p-0 flex-1 relative group">
              <div className="absolute left-0 top-0 bottom-0 w-12 bg-black/30 border-r border-border flex flex-col items-end py-4 pr-2 text-muted-foreground font-mono text-sm select-none opacity-50">
                {Array.from({length: 30}).map((_, i) => (
                  <div key={i} className="leading-6">{i + 1}</div>
                ))}
              </div>
              <textarea 
                className="w-full h-full bg-transparent text-sm font-mono text-gray-300 p-4 pl-16 resize-none focus:outline-none leading-6 font-medium"
                defaultValue={defaultConfig}
                spellCheck={false}
              />
            </CardContent>
          </Card>
      </div>
    </Layout>
  );
}
