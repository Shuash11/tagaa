import * as fs from "fs/promises";
import * as path from "path";
import { SessionLogEntry } from "../voting/schemas.js";

export interface SessionLog {
  sessionId: string;
  startedAt: string;
  completedAt: string | null;
  entries: SessionLogEntry[];
}

export interface SessionSummary {
  sessionId: string;
  startedAt: string;
  completedAt: string | null;
}

export class SessionLogger {
  private sessionsDir: string;
  private logPath: string;
  private log: SessionLog;

  constructor(sessionId: string, sessionsDir: string = "sessions") {
    this.sessionsDir = sessionsDir;
    this.logPath = path.join(sessionsDir, `session-${sessionId}.json`);
    this.log = {
      sessionId,
      startedAt: new Date().toISOString(),
      completedAt: null,
      entries: [],
    };
  }

  addEntry(entry: SessionLogEntry): void {
    this.log.entries.push(entry);
  }

  async save(): Promise<void> {
    this.log.completedAt = new Date().toISOString();
    await fs.mkdir(path.dirname(this.logPath), { recursive: true });
    await fs.writeFile(this.logPath, JSON.stringify(this.log, null, 2), "utf-8");
  }

  async load(): Promise<void> {
    try {
      const data = await fs.readFile(this.logPath, "utf-8");
      const parsed = JSON.parse(data);
      this.log = parsed;
    } catch {
      throw new Error(`Session log not found: ${this.logPath}`);
    }
  }

  getLog(): SessionLog {
    return this.log;
  }

  getLogPath(): string {
    return this.logPath;
  }

  async listSessions(): Promise<SessionSummary[]> {
    try {
      const entries = await fs.readdir(this.sessionsDir);
      const sessionFiles = entries.filter((e) => e.startsWith("session-") && e.endsWith(".json"));
      const summaries: SessionSummary[] = [];
      for (const file of sessionFiles) {
        try {
          const data = await fs.readFile(path.join(this.sessionsDir, file), "utf-8");
          const parsed = JSON.parse(data);
          const sessionId = file.replace(/^session-/, "").replace(/\.json$/, "");
          summaries.push({
            sessionId,
            startedAt: parsed.startedAt || "unknown",
            completedAt: parsed.completedAt || null,
          });
        } catch {
          // skip unreadable files
        }
      }
      summaries.sort((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime());
      return summaries;
    } catch {
      return [];
    }
  }
}