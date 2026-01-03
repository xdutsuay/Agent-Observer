import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useAgentSimulation } from "@/lib/simulation";
import { 
  Activity, 
  Cpu, 
  Box,
  Network
} from "lucide-react";
import { 
  AreaChart, 
  Area, 
  ResponsiveContainer, 
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid
} from "recharts";
import { cn } from "@/lib/utils";

const data = Array.from({ length: 40 }, (_, i) => ({
  time: i,
  val: 50 + Math.sin(i / 3) * 30 + Math.random() * 10
}));

export default function Dashboard() {
  const { events, metrics, isRunning } = useAgentSimulation();

  return (
    <Layout>
      <div className="p-10 space-y-10">
        {/* Header */}
        <div className="flex justify-between items-start">
          <div className="space-y-4">
            <h1 className="text-3xl font-bold tracking-tight text-white uppercase font-mono">
              DASHBOARD
            </h1>
            <p className="text-xs text-[#8b949e] font-mono tracking-wide">
              Real-time observation of active workspaces.
            </p>
          </div>
          
          <div className="flex gap-12 items-center font-mono">
            <div className="flex items-center gap-3">
               <span className="text-[11px] text-[#8b949e] uppercase font-bold tracking-widest">Agent Status</span>
               <div className="flex items-center gap-2 text-primary">
                 <div className="h-2 w-2 rounded-full bg-primary animate-pulse" />
                 <span className="text-[11px] font-black uppercase">ACTIVE</span>
               </div>
            </div>
            <div className="flex items-center gap-3">
               <span className="text-[11px] text-[#8b949e] uppercase font-bold tracking-widest">Confidence</span>
               <span className="text-primary text-sm font-black tracking-tighter">84.1%</span>
            </div>
          </div>
        </div>

        {/* Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard title="CPU LOAD" value="24%" icon={Cpu} trend="+2.4%" color="text-primary" />
          <MetricCard title="MEMORY" value="46.3 GB" icon={Box} trend="-0.1%" color="text-secondary" />
          <MetricCard title="EVENTS/SEC" value="24" icon={Activity} trend="+12%" color="text-green-500" />
          <MetricCard title="NETWORK" value="1.2 MB/s" icon={Network} trend="+5.5%" color="text-yellow-500" />
        </div>

        {/* Main Content Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Chart Section */}
          <div className="lg:col-span-8 flex flex-col space-y-4">
             <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.2em] uppercase px-1">
                ACTIVITY_HEURISTICS_V2
             </div>
             <Card className="bg-[#111317] border-[#21262d] rounded-md h-[400px] flex-1 overflow-hidden">
                <div className="h-full w-full p-4">
                   <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={data} margin={{ top: 20, right: 30, left: 0, bottom: 0 }}>
                        <defs>
                          <linearGradient id="colorWave" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="#00e5ff" stopOpacity={0.15}/>
                            <stop offset="95%" stopColor="#00e5ff" stopOpacity={0}/>
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" stroke="#21262d" vertical={false} />
                        <Area 
                          type="monotone" 
                          dataKey="val" 
                          stroke="#00e5ff" 
                          strokeWidth={2}
                          fillOpacity={1} 
                          fill="url(#colorWave)" 
                          animationDuration={2000}
                        />
                      </AreaChart>
                   </ResponsiveContainer>
                </div>
             </Card>
          </div>

          {/* Events Section */}
          <div className="lg:col-span-4 flex flex-col space-y-4">
             <div className="flex justify-between items-center text-[10px] text-[#8b949e] font-bold tracking-[0.2em] uppercase px-1">
                <span>LIVE_EVENTS</span>
                <Badge variant="outline" className="text-[9px] rounded-sm bg-white/5 border-[#21262d] py-0 text-white font-mono">REALTIME</Badge>
             </div>
             <Card className="bg-[#111317] border-[#21262d] rounded-md h-[400px] flex flex-col overflow-hidden">
                <CardContent className="p-0 flex-1 overflow-auto font-mono">
                   <div className="divide-y divide-[#21262d]">
                      {events.length > 0 ? events.map((event) => (
                        <div key={event.id} className="p-4 hover:bg-white/5 transition-all">
                           <div className="flex justify-between items-center mb-1">
                              <span className="text-[10px] font-bold text-primary">
                                [{event.severity || 'INFO'}]
                              </span>
                              <span className="text-[9px] text-[#8b949e]">
                                {new Date(event.timestamp).toLocaleTimeString([], { hour12: false })}
                              </span>
                           </div>
                           <p className="text-[11px] text-white leading-relaxed truncate">
                              {event.message}
                           </p>
                           <div className="mt-1 text-[8px] text-[#8b949e] uppercase">
                              {event.source || 'FILESYSTEMWATCHER'}
                           </div>
                        </div>
                      )) : (
                        <div className="p-10 text-center text-[#8b949e] text-xs font-mono italic">
                           NO_ACTIVITY_STREAM_DETECTED
                        </div>
                      )}
                   </div>
                </CardContent>
             </Card>
          </div>
        </div>
      </div>
    </Layout>
  );
}

function MetricCard({ title, value, icon: Icon, trend, color }: any) {
  return (
    <Card className="bg-[#111317] border-[#21262d] rounded-md group hover:border-primary/20 transition-all">
      <CardContent className="p-6">
        <div className="flex justify-between items-start mb-6">
           <div className="p-3 bg-[#1a1d23] rounded-md group-hover:bg-primary/5 transition-colors">
              <Icon className={cn("h-4 w-4", color)} />
           </div>
           <span className={cn("text-[10px] font-bold font-mono tracking-tighter", trend.startsWith('-') ? 'text-secondary' : 'text-green-500')}>
              {trend}
           </span>
        </div>
        <div className="space-y-1">
           <h3 className="text-[10px] font-bold text-[#8b949e] tracking-[0.1em] uppercase">{title}</h3>
           <p className="text-2xl font-black text-white tracking-tighter font-mono">{value}</p>
        </div>
      </CardContent>
    </Card>
  );
}
