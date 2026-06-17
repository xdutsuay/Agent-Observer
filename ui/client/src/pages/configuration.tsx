import { Layout } from "@/components/layout";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { useQuery } from "@tanstack/react-query";
import { HardDrive } from "lucide-react";
import { formatBytes } from "@/lib/utils";

export default function Configuration() {
  const { data: config } = useQuery<Record<string, unknown>>({
    queryKey: ["/api/config"],
  });
  const { data: status } = useQuery<{
    running: boolean;
    data_root: string;
    llm_provider: string;
    nvidia_configured?: boolean;
    watch_paths?: string[];
  }>({
    queryKey: ["/api/status"],
    refetchInterval: 3000,
  });

  const { data: disk } = useQuery<{
    data_root: string;
    overall: Record<string, string | number>;
    breakdown_human: Record<string, string>;
    projects: {
      name: string;
      memory_store_bytes_human: string;
      workspace_bytes_human: string | null;
    }[];
  }>({
    queryKey: ["/api/disk-usage"],
    refetchInterval: 60000,
  });

  const yaml = `# Agent Memory — live config
data_root: ${status?.data_root ?? "—"}
watcher_running: ${status?.running ?? false}
llm_provider: ${status?.llm_provider ?? "none"}
nvidia_configured: ${status?.nvidia_configured ?? false}

# NVIDIA (uses Hermes ~/.hermes/auth.json when AGENT_MEMORY_USE_HERMES_AUTH=1):
# AGENT_MEMORY_LLM_PROVIDER=nvidia

watch_paths:
${(status?.watch_paths ?? config?.watch_paths as string[] | undefined)?.map((p) => `  - ${p}`).join("\n") ?? "  - (loading)"}

process_name_contains:
${(config?.process_name_contains as string[] | undefined)?.map((p) => `  - ${p}`).join("\n") ?? ""}
`;

  return (
    <Layout>
      <div className="p-6 max-w-7xl mx-auto h-full flex flex-col">
        <header className="mb-8">
          <h2 className="text-2xl font-bold text-white mb-2">CONFIGURATION</h2>
          <p className="text-muted-foreground font-mono text-sm">
            Daemon settings from GET /api/config and /api/status
          </p>
        </header>

        {disk && (
          <Card className="mb-6 bg-[#111317] border-[#21262d]">
            <CardHeader className="border-b border-[#21262d] py-4">
              <CardTitle className="text-sm font-bold text-white flex items-center gap-2">
                <HardDrive className="h-4 w-4 text-primary" />
                DISK USAGE
              </CardTitle>
            </CardHeader>
            <CardContent className="p-5 space-y-4 font-mono text-[11px]">
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div>
                  <div className="text-[#8b949e] text-[9px] uppercase">Data root</div>
                  <div className="text-white">{disk.overall.data_root_bytes_human as string}</div>
                </div>
                <div>
                  <div className="text-[#8b949e] text-[9px] uppercase">Memory DB</div>
                  <div className="text-white">{formatBytes(disk.overall.memory_db_bytes as number)}</div>
                </div>
                <div>
                  <div className="text-[#8b949e] text-[9px] uppercase">Usage log DB</div>
                  <div className="text-white">{formatBytes(disk.overall.usage_db_bytes as number)}</div>
                </div>
                <div>
                  <div className="text-[#8b949e] text-[9px] uppercase">All workspaces</div>
                  <div className="text-white">{disk.overall.total_workspace_human as string}</div>
                </div>
              </div>
              <div className="border-t border-[#21262d] pt-3">
                <div className="text-[#8b949e] text-[9px] uppercase mb-2">Per project</div>
                <div className="space-y-1 max-h-40 overflow-auto">
                  {disk.projects.slice(0, 20).map((p) => (
                    <div key={p.name} className="flex justify-between text-[#8b949e]">
                      <span className="text-white/80">{p.name}</span>
                      <span>
                        mem {p.memory_store_bytes_human}
                        {p.workspace_bytes_human ? ` · disk ${p.workspace_bytes_human}` : ""}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        <Card className="flex-1 bg-card/50 border-border overflow-hidden flex flex-col">
          <CardHeader className="border-b border-border bg-black/20">
            <CardTitle className="text-xs font-mono text-muted-foreground">
              config.yaml (read-only)
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0 flex-1">
            <textarea
              className="w-full h-[500px] bg-transparent text-sm font-mono text-gray-300 p-4 resize-none focus:outline-none leading-6"
              value={yaml}
              readOnly
              spellCheck={false}
            />
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}
