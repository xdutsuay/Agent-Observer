import { Layout } from "@/components/layout";
import { useAgentSimulation, AgentEvent } from "@/lib/simulation";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Area, AreaChart, ResponsiveContainer, XAxis, YAxis, Tooltip } from "recharts";
import { Activity, HardDrive, Cpu, Network } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";

export default function Dashboard() {
  const { events, metrics } = useAgentSimulation();

  // Fake chart data
  const chartData = events.map((e, i) => ({
    name: i,
    value: Math.random() * 100 + 50
  })).reverse();

  return (
    <Layout>
      <div className="p-6 space-y-6 max-w-7xl mx-auto">
        <header className="flex justify-between items-center mb-8">
          <div>
            <h2 className="text-2xl font-bold text-white mb-2">DASHBOARD</h2>
            <p className="text-muted-foreground">Real-time observation of active workspaces.</p>
          </div>
          <div className="flex items-center gap-4">
            <div className="flex flex-col items-end">
              <span className="text-xs text-primary font-bold">AGENT STATUS</span>
              <span className="text-sm text-white flex items-center gap-2">
                <span className="h-2 w-2 bg-primary rounded-full animate-pulse" />
                ACTIVE
              </span>
            </div>
            <div className="h-10 w-[1px] bg-border" />
            <div className="flex flex-col items-end">
              <span className="text-xs text-muted-foreground font-bold">CONFIDENCE</span>
              <span className="text-xl font-mono text-primary">{metrics.confidence.toFixed(1)}%</span>
            </div>
          </div>
        </header>

        {/* Metrics Grid */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <MetricCard 
            title="CPU LOAD" 
            value={`${metrics.cpuLoad.toFixed(0)}%`} 
            icon={Cpu} 
            trend="+2.4%" 
            color="text-primary"
          />
          <MetricCard 
            title="MEMORY" 
            value={`${metrics.memoryUsage.toFixed(1)} GB`} 
            icon={HardDrive} 
            trend="-0.1%" 
            color="text-secondary"
          />
          <MetricCard 
            title="EVENTS/SEC" 
            value="24" 
            icon={Activity} 
            trend="+12%" 
            color="text-green-500"
          />
          <MetricCard 
            title="NETWORK" 
            value="1.2 MB/s" 
            icon={Network} 
            trend="+5.5%" 
            color="text-yellow-500"
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Main Activity Chart */}
          <Card className="lg:col-span-2 bg-card/50 border-border">
            <CardHeader>
              <CardTitle className="text-sm font-mono text-muted-foreground">ACTIVITY_HEURISTICS_V2</CardTitle>
            </CardHeader>
            <CardContent className="h-[300px]">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData}>
                  <defs>
                    <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="hsl(190 90% 50%)" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="hsl(190 90% 50%)" stopOpacity={0}/>
                    </linearGradient>
                  </defs>
                  <XAxis dataKey="name" hide />
                  <YAxis hide />
                  <Tooltip 
                    contentStyle={{ backgroundColor: '#09090b', borderColor: '#27272a', color: '#fff' }}
                    itemStyle={{ color: '#06b6d4' }}
                  />
                  <Area 
                    type="monotone" 
                    dataKey="value" 
                    stroke="hsl(190 90% 50%)" 
                    fillOpacity={1} 
                    fill="url(#colorValue)" 
                    isAnimationActive={false}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          {/* Live Event Log */}
          <Card className="bg-card/50 border-border flex flex-col h-[400px]">
            <CardHeader className="border-b border-border pb-3">
              <CardTitle className="text-sm font-mono text-muted-foreground flex justify-between items-center">
                <span>LIVE_EVENTS</span>
                <Badge variant="outline" className="font-mono text-[10px] h-5">REALTIME</Badge>
              </CardTitle>
            </CardHeader>
            <CardContent className="flex-1 overflow-auto p-0 scrollbar-hide">
              <div className="divide-y divide-border/50">
                <AnimatePresence initial={false}>
                  {events.map((event) => (
                    <motion.div
                      key={event.id}
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      exit={{ opacity: 0 }}
                      className="p-3 hover:bg-white/5 transition-colors font-mono text-xs"
                    >
                      <div className="flex justify-between mb-1">
                        <span className={event.severity === 'SUCCESS' ? 'text-green-500' : event.severity === 'WARNING' ? 'text-yellow-500' : 'text-primary'}>
                          [{event.severity}]
                        </span>
                        <span className="text-muted-foreground opacity-50">{event.timestamp.split('T')[1].split('.')[0]}</span>
                      </div>
                      <div className="text-white mb-1">{event.message}</div>
                      <div className="text-[10px] text-muted-foreground uppercase">{event.source}</div>
                    </motion.div>
                  ))}
                </AnimatePresence>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </Layout>
  );
}

function MetricCard({ title, value, icon: Icon, trend, color }: any) {
  return (
    <Card className="bg-card/50 border-border hover:border-primary/50 transition-colors group">
      <CardContent className="p-6">
        <div className="flex justify-between items-start mb-4">
          <div className="p-2 bg-white/5 rounded-none group-hover:bg-primary/10 transition-colors">
            <Icon className={`h-5 w-5 ${color}`} />
          </div>
          <span className={`text-xs font-mono ${trend.startsWith('+') ? 'text-green-500' : 'text-red-500'}`}>
            {trend}
          </span>
        </div>
        <div className="space-y-1">
          <h3 className="text-xs font-mono text-muted-foreground tracking-wider">{title}</h3>
          <div className="text-2xl font-bold font-mono text-white">{value}</div>
        </div>
      </CardContent>
    </Card>
  );
}
