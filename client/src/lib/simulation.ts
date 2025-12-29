import { useState, useEffect } from 'react';

export type EventType = 'PROCESS' | 'FILE' | 'NETWORK' | 'DECISION';
export type Severity = 'INFO' | 'WARNING' | 'CRITICAL' | 'SUCCESS';

export interface AgentEvent {
  id: string;
  timestamp: string;
  type: EventType;
  severity: Severity;
  message: string;
  source: string;
  metadata?: Record<string, any>;
}

const PROCESSES = ['python.exe', 'node', 'cargo', 'gopls', 'git'];
const FILES = ['/src/main.py', '/src/utils.ts', '/config/agent.yaml', '/logs/debug.log', '.git/index'];
const DECISIONS = ['Refactoring detected', 'Test failure analysis', 'Dependency injection', 'Commit message generation'];

export function useAgentSimulation() {
  const [events, setEvents] = useState<AgentEvent[]>([]);
  const [metrics, setMetrics] = useState({
    confidence: 85,
    cpuLoad: 12,
    memoryUsage: 45,
    activeAgents: 1
  });

  useEffect(() => {
    const interval = setInterval(() => {
      // Randomly generate events
      if (Math.random() > 0.6) {
        const type = Math.random() > 0.5 ? 'FILE' : (Math.random() > 0.5 ? 'PROCESS' : 'DECISION');
        let message = '';
        let source = '';
        let severity: Severity = 'INFO';

        if (type === 'FILE') {
          const file = FILES[Math.floor(Math.random() * FILES.length)];
          message = `File modification detected: ${file}`;
          source = 'FileSystemWatcher';
          severity = 'INFO';
        } else if (type === 'PROCESS') {
          const proc = PROCESSES[Math.floor(Math.random() * PROCESSES.length)];
          message = `High CPU usage detected on PID ${Math.floor(Math.random() * 9999)} (${proc})`;
          source = 'ProcessMonitor';
          severity = Math.random() > 0.8 ? 'WARNING' : 'INFO';
        } else {
          message = DECISIONS[Math.floor(Math.random() * DECISIONS.length)];
          source = 'HeuristicEngine';
          severity = 'SUCCESS';
        }

        const newEvent: AgentEvent = {
          id: Math.random().toString(36).substr(2, 9),
          timestamp: new Date().toISOString(),
          type: type as EventType,
          severity,
          message,
          source
        };

        setEvents(prev => [newEvent, ...prev].slice(0, 50));
      }

      // Update metrics slightly
      setMetrics(prev => ({
        ...prev,
        confidence: Math.min(100, Math.max(0, prev.confidence + (Math.random() - 0.5) * 5)),
        cpuLoad: Math.min(100, Math.max(0, prev.cpuLoad + (Math.random() - 0.5) * 10)),
        memoryUsage: Math.min(100, Math.max(0, prev.memoryUsage + (Math.random() - 0.5) * 2)),
      }));

    }, 1000);

    return () => clearInterval(interval);
  }, []);

  return { events, metrics };
}
