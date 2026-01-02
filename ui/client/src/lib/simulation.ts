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
  data_root: string;
  repos_count: number;
}

export function useAgentSimulation() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [events, setEvents] = useState<AgentEvent[]>([]);

  const { data: status } = useQuery<AgentStatus>({
    queryKey: ['/api/status'],
    refetchInterval: 2000,
  });

  const { data: reposData } = useQuery<{repos: Record<string, string>}>({
    queryKey: ['/api/repos'],
    refetchInterval: 5000,
  });

  const startMutation = useMutation({
    mutationFn: async () => {
      await apiRequest('POST', '/api/watcher/start');
    },
    onSuccess: () => {
      toast({ title: "Watcher Started", description: "The agent companion is now observing file changes." });
      queryClient.invalidateQueries({ queryKey: ['/api/status'] });
    },
    onError: (e) => {
      toast({ title: "Failed to start", description: String(e), variant: "destructive" });
    }
  });

  const stopMutation = useMutation({
    mutationFn: async () => {
      await apiRequest('POST', '/api/watcher/stop');
    },
    onSuccess: () => {
      toast({ title: "Watcher Stopped", description: "Observation halted." });
      queryClient.invalidateQueries({ queryKey: ['/api/status'] });
    },
    onError: (e) => {
      toast({ title: "Failed to stop", description: String(e), variant: "destructive" });
    }
  });

  const metrics = {
    confidence: status?.running ? 85 : 0,
    cpuLoad: 0,
    memoryUsage: 0,
    activeAgents: status?.running ? 1 : 0,
    reposCount: status?.repos_count || 0
  };

  return {
    events,
    metrics,
    repos: reposData?.repos || {},
    isRunning: status?.running,
    startAgent: () => startMutation.mutate(),
    stopAgent: () => stopMutation.mutate()
  };
}
