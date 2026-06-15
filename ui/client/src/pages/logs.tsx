import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useAgentSimulation } from "@/lib/simulation";
import { cn } from "@/lib/utils";
import { Terminal } from "lucide-react";

export default function SystemLogs() {
  const { events } = useAgentSimulation();
  const errors = events.filter((e) => e.severity === "ERROR");
  const info = events.filter((e) => e.severity !== "ERROR");

  return (
    <Layout>
      <div className="p-10 space-y-8 max-w-6xl">
        <header className="flex items-center gap-4">
          <Terminal className="h-8 w-8 text-primary" />
          <div>
            <h1 className="text-2xl font-black text-white uppercase font-mono">
              SYSTEM LOGS
            </h1>
            <p className="text-[10px] text-[#8b949e] font-mono tracking-widest">
              Live watcher stream — errors and file activity
            </p>
          </div>
        </header>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="space-y-3">
            <div className="text-[10px] text-secondary font-bold tracking-[0.2em] uppercase">
              ERRORS ({errors.length})
            </div>
            <Card className="bg-[#111317] border-[#21262d] h-[480px]">
              <ScrollArea className="h-full p-4">
                {errors.length === 0 ? (
                  <p className="text-[#8b949e] text-xs font-mono italic py-20 text-center">
                    No errors in the current stream.
                  </p>
                ) : (
                  <div className="space-y-3 font-mono text-[10px]">
                    {errors.map((e) => (
                      <div
                        key={e.id}
                        className="p-3 border border-secondary/20 rounded text-secondary"
                      >
                        <div className="flex justify-between mb-1">
                          <Badge variant="destructive" className="text-[8px]">
                            {e.source}
                          </Badge>
                          <span className="text-[#8b949e]">
                            {new Date(e.timestamp * 1000).toLocaleTimeString()}
                          </span>
                        </div>
                        <p>{e.message}</p>
                        {e.category && (
                          <p className="text-[#8b949e] mt-1 opacity-70">{e.category}</p>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </ScrollArea>
            </Card>
          </div>

          <div className="space-y-3">
            <div className="text-[10px] text-primary font-bold tracking-[0.2em] uppercase">
              ACTIVITY ({info.length})
            </div>
            <Card className="bg-[#111317] border-[#21262d] h-[480px]">
              <ScrollArea className="h-full p-4">
                <div className="space-y-2 font-mono text-[10px]">
                  {info.slice().reverse().map((e) => (
                    <div
                      key={e.id}
                      className={cn(
                        "p-2 border-b border-[#21262d] text-[#8b949e]",
                        "hover:bg-white/5"
                      )}
                    >
                      <span className="text-primary mr-2">
                        {new Date(e.timestamp * 1000).toLocaleTimeString()}
                      </span>
                      {e.message}
                    </div>
                  ))}
                  {info.length === 0 && (
                    <p className="italic py-20 text-center">
                      Start the daemon with watcher enabled to see activity.
                    </p>
                  )}
                </div>
              </ScrollArea>
            </Card>
          </div>
        </div>
      </div>
    </Layout>
  );
}
