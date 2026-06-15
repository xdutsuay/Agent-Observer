import { Layout } from "@/components/layout";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { useQuery } from "@tanstack/react-query";

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
