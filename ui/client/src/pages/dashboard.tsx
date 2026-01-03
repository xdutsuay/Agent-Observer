import { Layout } from "@/components/layout";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAgentSimulation } from "@/lib/simulation";
import { 
  Activity, 
  Cpu, 
  Database, 
  Play, 
  Square, 
  Zap,
  Box,
  Layers,
  ShieldCheck
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";

export default function Dashboard() {
  const { events, metrics, isRunning, startAgent, stopAgent } = useAgentSimulation();

  return (
    <Layout>
      <div className="p-8 max-w-7xl mx-auto space-y-8">
        <header className="flex justify-between items-end mb-4">
          <motion.div 
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
          >
            <h1 className="text-4xl font-extrabold tracking-tighter text-white mb-2 bg-clip-text text-transparent bg-gradient-to-r from-primary via-secondary to-accent">
              AGENT_CORE // V2
            </h1>
            <p className="text-muted-foreground font-mono flex items-center gap-2">
              <span className="h-2 w-2 rounded-full bg-primary animate-pulse" />
              SYSTEM PROTOCOL STACK: ACTIVE
            </p>
          </motion.div>
          
          <motion.div 
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            className="flex gap-4"
          >
            <Button
              variant={isRunning ? "destructive" : "default"}
              onClick={isRunning ? stopAgent : startAgent}
              className={`h-12 px-8 font-bold tracking-widest glass-panel hover:bg-white/10 transition-all duration-500 rounded-none ${isRunning ? 'text-red-400 border-red-900/50' : 'text-primary border-primary/50'}`}
            >
              {isRunning ? (
                <>
                  <Square className="mr-2 h-4 w-4 fill-current" /> STOP_KERNEL
                </>
              ) : (
                <>
                  <Play className="mr-2 h-4 w-4 fill-current" /> START_OBSERVER
                </>
              )}
            </Button>
          </motion.div>
        </header>

        {/* Intelligence Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            title="Intelligence Confidence"
            value={`${metrics.confidence}%`}
            icon={Zap}
            trend="+12.4%"
            color="text-primary"
            delay={0}
          />
          <MetricCard
            title="Kernel Threads"
            value={metrics.activeAgents}
            icon={Layers}
            trend="STABLE"
            color="text-secondary"
            delay={0.1}
          />
          <MetricCard
            title="Memory Buffers"
            value={metrics.reposCount}
            icon={Box}
            trend="+2"
            color="text-accent"
            delay={0.2}
          />
          <MetricCard
            title="Process Integrity"
            value="OK"
            icon={ShieldCheck}
            trend="100%"
            color="text-green-400"
            delay={0.3}
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Main Visualizer */}
          <Card className="lg:col-span-2 glass-panel rounded-none overflow-hidden relative border-none">
             <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-accent/5" />
             <CardHeader className="border-b border-white/5 relative z-10">
               <CardTitle className="text-xs font-mono uppercase tracking-[0.2em] text-muted-foreground flex items-center gap-2">
                 <Activity className="h-4 w-4 text-primary" />
                 Neural_Activity_Broadcasting
               </CardTitle>
             </CardHeader>
             <CardContent className="h-[400px] flex items-center justify-center relative z-10">
                <div className="text-center space-y-4">
                   <motion.div 
                     animate={{ 
                       scale: [1, 1.1, 1],
                       opacity: [0.5, 1, 0.5]
                     }}
                     transition={{ duration: 4, repeat: Infinity }}
                     className="h-48 w-48 rounded-full border-4 border-primary/20 flex items-center justify-center p-8 bg-black/40 backdrop-blur-3xl"
                   >
                     <Cpu className="h-full w-full text-primary/40" />
                   </motion.div>
                   <p className="font-mono text-sm text-muted-foreground animate-pulse text-primary shadow-primary/50 drop-shadow-lg">
                      {isRunning ? "AWAITING_INPUT_STREAM" : "KERNEL_IDLE"}
                   </p>
                </div>
             </CardContent>
          </Card>

          {/* Real-time Telemetry */}
          <Card className="glass-panel rounded-none flex flex-col border-none">
            <CardHeader className="border-b border-white/5">
              <CardTitle className="text-xs font-mono uppercase tracking-[0.2em] text-muted-foreground flex justify-between items-center">
                <span>TELEMETRY_LOG</span>
                <Badge variant="outline" className="font-mono text-[10px] h-5 bg-white/5 text-primary border-primary/20">LIVE</Badge>
              </CardTitle>
            </CardHeader>
            <CardContent className="flex-1 overflow-auto p-0 font-mono">
              <div className="divide-y divide-white/5">
                <AnimatePresence initial={false}>
                  {events.length > 0 ? events.map((event) => (
                    <motion.div
                      key={event.id}
                      initial={{ opacity: 0, x: 20 }}
                      animate={{ opacity: 1, x: 0 }}
                      exit={{ opacity: 0 }}
                      className="p-4 hover:bg-white/5 transition-colors"
                    >
                      <div className="flex justify-between mb-2">
                        <span className={`text-[10px] px-1 bg-white/5 ${
                          event.severity === 'SUCCESS' ? 'text-green-400' : 
                          event.severity === 'WARNING' ? 'text-yellow-400' : 'text-primary'
                        }`}>
                          {event.severity}
                        </span>
                        <span className="text-[10px] text-muted-foreground opacity-50">{event.timestamp.split('T')[1]?.split('.')[0] || event.timestamp}</span>
                      </div>
                      <div className="text-xs text-white leading-relaxed">{event.message}</div>
                    </motion.div>
                  )) : (
                    <div className="p-8 text-center text-muted-foreground text-xs italic">
                      NO_TELEMETRY_DATA_DETECTED
                    </div>
                  )}
                </AnimatePresence>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </Layout>
  );
}

function MetricCard({ title, value, icon: Icon, trend, color, delay }: any) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay }}
    >
      <Card className="glass-panel border-none rounded-none hover:bg-white/5 transition-all duration-500 group">
        <CardContent className="p-6">
          <div className="flex justify-between items-start mb-6">
            <div className="p-3 bg-white/5 rounded-none group-hover:bg-primary/10 transition-colors">
              <Icon className={`h-6 w-6 ${color}`} />
            </div>
            <span className={`text-[10px] font-mono font-bold tracking-widest ${trend.startsWith('+') ? 'text-green-400' : 'text-muted-foreground'}`}>
              {trend}
            </span>
          </div>
          <div className="space-y-1">
            <h3 className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">{title}</h3>
            <div className="text-3xl font-black font-mono text-white tracking-tighter">{value}</div>
          </div>
        </CardContent>
      </Card>
    </motion.div>
  );
}
