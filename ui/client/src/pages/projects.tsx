import { Layout } from "@/components/layout";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useQuery } from "@tanstack/react-query";
import { FolderOpen, AlertTriangle, Code, Layers, HardDrive } from "lucide-react";
import { cn, formatBytes } from "@/lib/utils";

interface ProjectInfo {
  repo_id: string;
  path: string;
  name: string;
  language: string | null;
  framework: string | null;
  last_activity: number | null;
  memory_count: number;
  failure_count: number;
  tags: string[];
}

interface DiskProject {
  repo_id: string;
  memory_store_bytes: number;
  memory_store_bytes_human: string;
  workspace_bytes: number | null;
  workspace_bytes_human: string | null;
}

interface DiskReport {
  overall: {
    data_root_bytes_human: string;
    total_memory_attributed_human: string;
    total_workspace_human: string;
    memory_db_bytes: number;
    usage_db_bytes: number;
  };
  breakdown_human: Record<string, string>;
  projects: DiskProject[];
}

export default function Projects() {
  const { data } = useQuery<{ projects: ProjectInfo[] }>({
    queryKey: ["/api/projects"],
    refetchInterval: 10000,
  });

  const { data: disk } = useQuery<DiskReport>({
    queryKey: ["/api/disk-usage"],
    refetchInterval: 60000,
  });

  const diskByRepo = new Map(
    (disk?.projects || []).map((p) => [p.repo_id, p])
  );

  const projects = data?.projects || [];
  const active = projects.filter((p) => p.memory_count > 0);
  const inactive = projects.filter((p) => p.memory_count === 0);

  return (
    <Layout>
      <div className="p-8 space-y-8">
        <header>
          <div className="flex items-center gap-3 mb-1">
            <FolderOpen className="h-6 w-6 text-primary" />
            <h1 className="text-2xl font-black text-white uppercase">Projects</h1>
          </div>
          <p className="text-xs text-[#8b949e]">
            {projects.length} tracked projects — auto-detected from ~/localcode
          </p>
        </header>

        {disk && (
          <Card className="bg-[#111317] border-[#21262d]">
            <CardContent className="p-5 grid grid-cols-2 md:grid-cols-4 gap-4">
              <DiskStat label="Agent data" value={disk.overall.data_root_bytes_human} />
              <DiskStat label="Memory DB" value={formatBytes(disk.overall.memory_db_bytes)} />
              <DiskStat label="Memory stored" value={disk.overall.total_memory_attributed_human} />
              <DiskStat label="Workspaces (total)" value={disk.overall.total_workspace_human} />
            </CardContent>
          </Card>
        )}

        {/* Active projects */}
        {active.length > 0 && (
          <div className="space-y-3">
            <div className="text-[10px] text-primary font-bold tracking-[0.15em] uppercase px-1">
              Active ({active.length})
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {active.map((p) => (
                <ProjectCard key={p.repo_id} project={p} disk={diskByRepo.get(p.repo_id)} />
              ))}
            </div>
          </div>
        )}

        {/* Inactive projects */}
        {inactive.length > 0 && (
          <div className="space-y-3">
            <div className="text-[10px] text-[#8b949e] font-bold tracking-[0.15em] uppercase px-1">
              No memories yet ({inactive.length})
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {inactive.map((p) => (
                <ProjectCard key={p.repo_id} project={p} disk={diskByRepo.get(p.repo_id)} />
              ))}
            </div>
          </div>
        )}
      </div>
    </Layout>
  );
}

function DiskStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-2">
      <HardDrive className="h-4 w-4 text-primary shrink-0" />
      <div>
        <div className="text-[9px] text-[#8b949e] uppercase tracking-widest">{label}</div>
        <div className="text-sm font-bold text-white">{value}</div>
      </div>
    </div>
  );
}

function ProjectCard({
  project: p,
  disk,
}: {
  project: ProjectInfo;
  disk?: DiskProject;
}) {
  const hasFails = p.failure_count > 0;

  return (
    <Card className={cn(
      "bg-[#111317] border-[#21262d] hover:border-primary/30 transition-all",
      hasFails && "border-l-2 border-l-orange-500"
    )}>
      <CardContent className="p-5 space-y-4">
        <div className="flex items-start justify-between">
          <div>
            <h3 className="text-sm font-bold text-white">{p.name}</h3>
            <div className="text-[10px] text-[#8b949e] mt-0.5">{p.repo_id}</div>
          </div>
          {hasFails ? (
            <AlertTriangle className="h-4 w-4 text-orange-400" />
          ) : (
            <div className="h-2 w-2 rounded-full bg-green-400 mt-1" />
          )}
        </div>

        <div className="flex gap-2 flex-wrap">
          {p.language && (
            <Badge variant="outline" className="text-[9px] bg-primary/5 border-primary/20 text-primary">
              <Code className="h-2.5 w-2.5 mr-1" />
              {p.language}
            </Badge>
          )}
          {p.framework && (
            <Badge variant="outline" className="text-[9px] bg-blue-500/5 border-blue-500/20 text-blue-400">
              <Layers className="h-2.5 w-2.5 mr-1" />
              {p.framework}
            </Badge>
          )}
          {p.tags.filter((t) => t !== p.language && t !== p.framework).map((tag) => (
            <Badge key={tag} variant="outline" className="text-[9px] border-[#21262d] text-[#8b949e]">
              {tag}
            </Badge>
          ))}
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2 text-[10px] text-[#8b949e] pt-2 border-t border-[#21262d]">
          <span>{p.memory_count} memories</span>
          {disk && (
            <span className="text-primary font-mono" title="Memory in agent DB · workspace on disk">
              mem {disk.memory_store_bytes_human}
              {disk.workspace_bytes_human ? ` · disk ${disk.workspace_bytes_human}` : ""}
            </span>
          )}
          {hasFails && (
            <span className="text-orange-400 font-bold">{p.failure_count} failures</span>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
