import { useState } from "react";
import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useQuery } from "@tanstack/react-query";
import {
  Plug,
  Search,
  PenLine,
  ChevronDown,
  ChevronRight,
  Monitor,
  Activity,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface UsageSummary {
  total_interactions: number;
  last_24h: number;
  reads: number;
  writes: number;
  by_method: { method: string; count: number }[];
  by_host_ide: { host_ide: string; count: number }[];
  by_transport: { transport: string; count: number }[];
  running_ides: { label: string; process_count: number }[];
}

interface Session {
  id: string;
  client_name: string;
  client_version: string;
  host_ide: string;
  transport: string;
  connected_at: string;
  last_seen_at: string;
  call_count: number;
  last_call: string | null;
}

interface Interaction {
  id: string;
  transport: string;
  method: string;
  client_name: string;
  host_ide: string;
  query_summary: string;
  query_json: string;
  response_preview: string;
  duration_ms: number;
  ok: number;
  created_at: string;
}

function hostColor(host: string) {
  const map: Record<string, string> = {
    Cursor: "text-primary border-primary/30",
    "Claude Code": "text-orange-400 border-orange-400/30",
    "VS Code": "text-blue-400 border-blue-400/30",
    "Web UI": "text-purple-400 border-purple-400/30",
    Zed: "text-green-400 border-green-400/30",
  };
  return map[host] || "text-[#8b949e] border-[#21262d]";
}

function InteractionRow({ row }: { row: Interaction }) {
  const [open, setOpen] = useState(false);
  let queryObj: Record<string, unknown> = {};
  try {
    queryObj = JSON.parse(row.query_json || "{}");
  } catch {
    queryObj = {};
  }

  return (
    <div className="border border-[#21262d] rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full flex items-start gap-3 p-3 text-left hover:bg-white/5 transition-colors"
      >
        {open ? (
          <ChevronDown className="h-3.5 w-3.5 mt-0.5 shrink-0 text-[#8b949e]" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 mt-0.5 shrink-0 text-[#8b949e]" />
        )}
        <div className="flex-1 min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <Badge
              variant="outline"
              className={cn("text-[8px] font-bold", hostColor(row.host_ide))}
            >
              {row.host_ide}
            </Badge>
            <Badge variant="secondary" className="text-[8px]">
              {row.transport.toUpperCase()}
            </Badge>
            <span className="text-[10px] font-bold text-white">{row.method}</span>
            {!row.ok && (
              <Badge variant="destructive" className="text-[8px]">
                ERR
              </Badge>
            )}
            <span className="text-[9px] text-[#8b949e] ml-auto">
              {new Date(row.created_at).toLocaleString()}
            </span>
          </div>
          <p className="text-[10px] text-primary font-mono truncate">{row.query_summary}</p>
          <p className="text-[10px] text-[#8b949e] font-mono line-clamp-2">
            → {row.response_preview || "(empty)"}
          </p>
        </div>
      </button>
      {open && (
        <div className="px-4 pb-4 pt-0 space-y-3 border-t border-[#21262d] bg-black/20">
          <div>
            <div className="text-[9px] text-[#8b949e] uppercase tracking-widest mb-1">
              Query
            </div>
            <pre className="text-[10px] font-mono text-white/80 whitespace-pre-wrap break-all bg-[#0d0f14] p-2 rounded">
              {JSON.stringify(queryObj, null, 2)}
            </pre>
          </div>
          <div>
            <div className="text-[9px] text-[#8b949e] uppercase tracking-widest mb-1">
              Response
            </div>
            <pre className="text-[10px] font-mono text-white/80 whitespace-pre-wrap break-all bg-[#0d0f14] p-2 rounded max-h-48 overflow-auto">
              {row.response_preview || "(no response body)"}
            </pre>
          </div>
          <div className="text-[9px] text-[#8b949e]">
            Client: {row.client_name} · {row.duration_ms.toFixed(1)} ms
          </div>
        </div>
      )}
    </div>
  );
}

export default function UsagePage() {
  const { data: summary } = useQuery<UsageSummary>({
    queryKey: ["/api/usage/summary"],
    refetchInterval: 5000,
  });

  const { data: sessionsData } = useQuery<{ sessions: Session[] }>({
    queryKey: ["/api/usage/sessions"],
    refetchInterval: 5000,
  });

  const { data: interactionsData } = useQuery<{ interactions: Interaction[] }>({
    queryKey: ["/api/usage/interactions"],
    refetchInterval: 3000,
  });

  const sessions = sessionsData?.sessions || [];
  const interactions = interactionsData?.interactions || [];
  const running = summary?.running_ides || [];

  return (
    <Layout>
      <div className="p-10 space-y-8 max-w-6xl">
        <header className="flex items-center gap-4">
          <Plug className="h-8 w-8 text-primary" />
          <div>
            <h1 className="text-2xl font-black text-white uppercase font-mono">
              AGENT USAGE
            </h1>
            <p className="text-[10px] text-[#8b949e] font-mono tracking-widest">
              MCP + HTTP queries — who connected, what was asked, what came back
            </p>
          </div>
        </header>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[
            { label: "TOTAL CALLS", value: summary?.total_interactions ?? 0, icon: Activity },
            { label: "LAST 24H", value: summary?.last_24h ?? 0, icon: Activity },
            { label: "READS", value: summary?.reads ?? 0, icon: Search },
            { label: "WRITES", value: summary?.writes ?? 0, icon: PenLine },
          ].map((s) => (
            <Card key={s.label} className="bg-[#111317] border-[#21262d]">
              <CardContent className="p-4 flex items-center gap-3">
                <s.icon className="h-4 w-4 text-primary" />
                <div>
                  <div className="text-[9px] text-[#8b949e] tracking-widest">{s.label}</div>
                  <div className="text-xl font-black text-white">{s.value}</div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <Card className="bg-[#111317] border-[#21262d] lg:col-span-1">
            <CardContent className="p-5 space-y-4">
              <div className="flex items-center gap-2">
                <Monitor className="h-4 w-4 text-primary" />
                <span className="text-[10px] font-bold tracking-widest text-white">
                  RUNNING ON MACHINE
                </span>
              </div>
              {running.length === 0 ? (
                <p className="text-[10px] text-[#8b949e] font-mono italic">
                  No known IDEs detected in process list.
                </p>
              ) : (
                <div className="space-y-2">
                  {running.map((ide) => (
                    <div
                      key={ide.label}
                      className="flex justify-between items-center p-2 rounded border border-[#21262d]"
                    >
                      <span className={cn("text-[11px] font-bold", hostColor(ide.label).split(" ")[0])}>
                        {ide.label}
                      </span>
                      <Badge variant="outline" className="text-[8px]">
                        {ide.process_count} proc
                      </Badge>
                    </div>
                  ))}
                </div>
              )}

              <div className="pt-2 border-t border-[#21262d]">
                <div className="text-[9px] text-[#8b949e] uppercase tracking-widest mb-2">
                  Connected clients
                </div>
                {sessions.length === 0 ? (
                  <p className="text-[10px] text-[#8b949e] italic">
                    No MCP/HTTP sessions yet. Connect via Cursor MCP or search in the UI.
                  </p>
                ) : (
                  <div className="space-y-2">
                    {sessions.slice(0, 8).map((s) => (
                      <div key={s.id} className="text-[10px] font-mono">
                        <div className="flex justify-between">
                          <span className={hostColor(s.host_ide).split(" ")[0]}>
                            {s.host_ide}
                          </span>
                          <span className="text-[#8b949e]">{s.call_count} calls</span>
                        </div>
                        <div className="text-[#8b949e] truncate">{s.client_name}</div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {summary?.by_method && summary.by_method.length > 0 && (
                <div className="pt-2 border-t border-[#21262d]">
                  <div className="text-[9px] text-[#8b949e] uppercase tracking-widest mb-2">
                    Top tools
                  </div>
                  {summary.by_method.slice(0, 6).map((m) => (
                    <div
                      key={m.method}
                      className="flex justify-between text-[10px] font-mono py-0.5"
                    >
                      <span className="text-white/80">{m.method}</span>
                      <span className="text-primary">{m.count}</span>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          <Card className="bg-[#111317] border-[#21262d] lg:col-span-2">
            <CardContent className="p-5">
              <div className="text-[10px] font-bold tracking-widest text-white mb-4">
                QUERY / RESPONSE LOG
              </div>
              <ScrollArea className="h-[520px] pr-3">
                {interactions.length === 0 ? (
                  <p className="text-[10px] text-[#8b949e] font-mono italic py-16 text-center">
                    No interactions logged yet.
                    <br />
                    Use MCP tools from Cursor or Claude Code, or search memory in the UI.
                  </p>
                ) : (
                  <div className="space-y-2">
                    {interactions.map((row) => (
                      <InteractionRow key={row.id} row={row} />
                    ))}
                  </div>
                )}
              </ScrollArea>
            </CardContent>
          </Card>
        </div>
      </div>
    </Layout>
  );
}
