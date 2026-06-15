import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiRequest } from "./queryClient";

export interface AgentEvent {
  id: string;
  timestamp: number;
  message: string;
  source: string;
  severity: string;
  category?: string;
}

export interface SystemMetrics {
  cpu_percent: number;
  memory_used_gb: number;
  memory_total_gb: number;
  network_mbps: number;
  fs_events_per_sec: number;
  active_processes: number;
  agent_processes_detected: number;
  activity_score: number;
  timestamp: number;
}

export interface Repo {
  id: string;
  path: string;
  error_count: number;
  health: string;
  last_modified: string;
}

export function useAgentSimulation() {
  const queryClient = useQueryClient();

  // Fetch watcher status
  const { data: statusData } = useQuery({
    queryKey: ["/api/status"],
    refetchInterval: 2000,
  });

  // Fetch repos
  const { data: reposData } = useQuery({
    queryKey: ["/api/repos"],
    refetchInterval: 5000,
  });

  // Fetch real-time metrics
  const { data: metricsData } = useQuery<SystemMetrics>({
    queryKey: ["/api/metrics"],
    refetchInterval: 1000, // Update every second for real-time feel
  });

  // Fetch live events
  const { data: eventsData } = useQuery<{ events: AgentEvent[] }>({
    queryKey: ["/api/events"],
    refetchInterval: 2000,
  });

  // Start watcher mutation
  const startMutation = useMutation({
    mutationFn: () => apiRequest("POST", "/api/watcher/start"),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/api/status"] });
    },
  });

  // Stop watcher mutation
  const stopMutation = useMutation({
    mutationFn: () => apiRequest("POST", "/api/watcher/stop"),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/api/status"] });
    },
  });

  const isRunning = statusData?.running || false;
  const repos: Repo[] = reposData?.repos || [];
  const metrics = metricsData || {
    cpu_percent: 0,
    memory_used_gb: 0,
    memory_total_gb: 0,
    network_mbps: 0,
    fs_events_per_sec: 0,
    active_processes: 0,
    agent_processes_detected: 0,
    activity_score: 0,
    timestamp: Date.now() / 1000
  };
  const events = eventsData?.events || [];

  return {
    isRunning,
    repos,
    metrics,
    events,
    start: () => startMutation.mutate(),
    stop: () => stopMutation.mutate(),
  };
}
