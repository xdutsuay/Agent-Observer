import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiRequest } from './queryClient';
import { useToast } from "@/hooks/use-toast";

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

export interface AgentStatus {
  running: boolean;
  score: number;
  ts: number;
}

export function useAgentSimulation() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [events, setEvents] = useState<AgentEvent[]>([]);

  const { data: status } = useQuery<AgentStatus>({
    queryKey: ['/api/agent/status'],
    refetchInterval: 1000,
  });

  const startmutation = useMutation({
    mutationFn: async () => {
      await apiRequest('POST', '/api/agent/start');
    },
    onSuccess: () => {
      toast({ title: "Agent Started", description: "The agent companion is now observing." });
      queryClient.invalidateQueries({ queryKey: ['/api/agent/status'] });
    },
    onError: (e) => {
      toast({ title: "Failed to start", description: String(e), variant: "destructive" });
    }
  });

  const stopMutation = useMutation({
    mutationFn: async () => {
      await apiRequest('POST', '/api/agent/stop');
    },
    onSuccess: () => {
      toast({ title: "Agent Stopped", description: "Observation halted." });
      queryClient.invalidateQueries({ queryKey: ['/api/agent/status'] });
    },
    onError: (e) => {
      toast({ title: "Failed to stop", description: String(e), variant: "destructive" });
    }
  });

  // Transform status into metrics format expected by UI
  const metrics = {
    confidence: (status?.score || 0) * 100, // Assuming score is 0-1
    cpuLoad: 0, // Not available in simple status yet, needing psutil in backend/CLI to expose more
    memoryUsage: 0,
    activeAgents: status?.running ? 1 : 0
  };

  return {
    events,
    metrics,
    isRunning: status?.running,
    startAgent: startmutation.mutate,
    stopAgent: stopMutation.mutate
  };
}
