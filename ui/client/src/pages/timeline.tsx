import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useQuery } from "@tanstack/react-query";
import { Clock, Zap, AlertTriangle, CheckCircle, FolderOpen, FileText, Eye } from "lucide-react";
import { cn } from "@/lib/utils";
import { useState, useEffect, useRef } from "react";

interface TimelineEvent {
  event_type: string;
  data: Record<string, any>;
  timestamp: number;
}

const EVENT_CONFIG: Record<string, { icon: any; color: string; label: string }> = {
  memory_created: { icon: Zap, color: "text-primary", label: "Memory Created" },
  memory_deleted: { icon: FileText, color: "text-[#8b949e]", label: "Memory Deleted" },
  failure_detected: { icon: AlertTriangle, color: "text-orange-400", label: "Failure Detected" },
  failure_resolved: { icon: CheckCircle, color: "text-green-400", label: "Failure Resolved" },
  file_changed: { icon: FileText, color: "text-blue-400", label: "File Changed" },
  repo_discovered: { icon: FolderOpen, color: "text-purple-400", label: "Repo Discovered" },
  watcher_status: { icon: Eye, color: "text-yellow-400", label: "Watcher Status" },
};

export default function Timeline() {
  const [wsEvents, setWsEvents] = useState<TimelineEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  // Fetch history on mount
  const { data: historyData } = useQuery<{ events: TimelineEvent[] }>({
    queryKey: ["/api/events/history"],
    refetchOnWindowFocus: false,
  });

  // WebSocket for live events
  useEffect(() => {
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(`${proto}//${window.location.host}/ws/events`);
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);
    ws.onmessage = (msg) => {
      try {
        const evt = JSON.parse(msg.data);
        setWsEvents((prev) => [evt, ...prev].slice(0, 200));
      } catch {}
    };

    return () => ws.close();
  }, []);

  const historyEvents = (historyData?.events || []).sort((a, b) => b.timestamp - a.timestamp);
  const allEvents = [...wsEvents, ...historyEvents];

  // Dedupe by timestamp+type
  const seen = new Set<string>();
  const events = allEvents.filter((e) => {
    const key = `${e.event_type}-${e.timestamp}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  }).slice(0, 200);

  return (
    <Layout>
      <div className="p-8 space-y-6">
        <header className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-3 mb-1">
              <Clock className="h-6 w-6 text-primary" />
              <h1 className="text-2xl font-black text-white uppercase">Timeline</h1>
            </div>
            <p className="text-xs text-[#8b949e]">
              Real-time event stream via WebSocket
            </p>
          </div>
          <Badge
            variant="outline"
            className={cn(
              "text-[9px] font-bold",
              connected
                ? "text-green-400 border-green-400/30 bg-green-400/5"
                : "text-[#8b949e] border-[#21262d]"
            )}
          >
            {connected ? "● LIVE" : "○ DISCONNECTED"}
          </Badge>
        </header>

        {/* Event stream */}
        <Card className="bg-[#111317] border-[#21262d]">
          <CardContent className="p-0 max-h-[calc(100vh-200px)] overflow-auto">
            {events.length === 0 ? (
              <div className="p-12 text-center text-[#8b949e] text-xs">
                No events yet — start the watcher or use MCP tools to generate events
              </div>
            ) : (
              <div className="divide-y divide-[#21262d]">
                {events.map((evt, i) => {
                  const config = EVENT_CONFIG[evt.event_type] || {
                    icon: Zap,
                    color: "text-[#8b949e]",
                    label: evt.event_type,
                  };
                  const Icon = config.icon;
                  const ts = new Date(evt.timestamp * 1000);

                  return (
                    <div key={`${evt.event_type}-${evt.timestamp}-${i}`} className="px-5 py-3 hover:bg-white/5 flex items-start gap-4">
                      {/* Timeline dot */}
                      <div className="flex flex-col items-center mt-1">
                        <div className={cn("p-1.5 rounded bg-[#1a1d23]", config.color)}>
                          <Icon className="h-3 w-3" />
                        </div>
                        {i < events.length - 1 && (
                          <div className="w-px h-6 bg-[#21262d] mt-1" />
                        )}
                      </div>

                      {/* Content */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-0.5">
                          <span className={cn("text-[11px] font-bold", config.color)}>
                            {config.label}
                          </span>
                          {evt.data.repo_id && (
                            <Badge variant="outline" className="text-[8px] border-[#21262d] text-[#8b949e]">
                              {evt.data.repo_id}
                            </Badge>
                          )}
                        </div>
                        {evt.data.content && (
                          <div className="text-[10px] text-white truncate max-w-lg">
                            {evt.data.content}
                          </div>
                        )}
                        {evt.data.signature && (
                          <div className="text-[10px] text-orange-300 font-mono truncate">
                            {evt.data.signature}
                          </div>
                        )}
                        {evt.data.path && (
                          <div className="text-[10px] text-[#8b949e] font-mono truncate">
                            {evt.data.path}
                          </div>
                        )}
                      </div>

                      {/* Timestamp */}
                      <div className="text-[9px] text-[#8b949e] shrink-0 mt-0.5">
                        {ts.toLocaleTimeString([], { hour12: false })}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}
