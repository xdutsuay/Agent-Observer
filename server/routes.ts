import type { Express } from "express";
import { createServer, type Server } from "http";
import { storage } from "./storage";

export async function registerRoutes(
  httpServer: Server,
  app: Express
): Promise<Server> {
  const { spawn, exec } = await import("child_process");
  const fs = await import("fs/promises");
  const path = await import("path");
  const os = await import("os");

  let agentProcess: any = null;

  app.post("/api/agent/start", (req, res) => {
    if (agentProcess) {
      return res.status(400).json({ message: "Agent already running" });
    }

    const cliPath = path.resolve(process.cwd(), "agent_companion_cli");
    const venvPython = path.join(cliPath, ".venv/bin/python");

    // We run the module directly
    agentProcess = spawn(venvPython, ["-m", "agent_companion_cli", "start"], {
      cwd: cliPath,
      stdio: "ignore", // simple background run
      detached: true // ensure it keeps running
    });

    agentProcess.unref();

    res.json({ status: "started" });
  });

  app.post("/api/agent/stop", (req, res) => {
    if (agentProcess) {
      agentProcess.kill();
      agentProcess = null;
      res.json({ status: "stopped" });
    } else {
      // It might be running but we lost reference, force kill by name? 
      // For now just say not running.
      // Or we can try pkill logic if needed, but let's stick to simple process tracking for this session.
      res.json({ status: "stopped", message: "No active process found managed by server" });
    }
  });

  app.get("/api/agent/status", async (req, res) => {
    try {
      const homeDir = os.homedir();
      const statusPath = path.join(homeDir, "agent_companion_data", "runtime", "status.json");

      try {
        const content = await fs.readFile(statusPath, "utf-8");
        const data = JSON.parse(content);
        res.json({
          running: !!agentProcess,
          ...data
        });
      } catch (e) {
        // File might not exist yet
        res.json({ running: !!agentProcess, score: 0, ts: 0 });
      }
    } catch (error) {
      res.status(500).json({ message: "Internal Error" });
    }
  });

  return httpServer;
}
