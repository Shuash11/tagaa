import crypto from "crypto";
import { AgentPool, buildAgentPoolFromConfig } from "../agents/pool.js";
import { Agent } from "../agents/agent.js";
import { Config } from "../voting/schemas.js";
import { ToolLayer, createToolLayer } from "../tools/index.js";
import { SessionLogger } from "../logger/session-logger.js";
import { SessionState, createSessionState } from "./state.js";
import { runPhase1Intake } from "../phases/phase1-intake.js";
import { runPhase2Plan } from "../phases/phase2-plan.js";
import { runPhase3PlanVote } from "../phases/phase3-plan-vote.js";
import { runPhase4CompatVote } from "../phases/phase4-compat-vote.js";
import { runPhase5Execute } from "../phases/phase5-execute.js";
import { messageStore, createMessage } from "../ui/message-store.js";
import { appState } from "../ui/app-state.js";

export class Orchestrator {
  private pool: AgentPool;
  private config: Config;
  private toolLayer: ToolLayer;
  private state: SessionState;
  private logger: SessionLogger;
  private isRunning: boolean = false;

  constructor(config: Config) {
    this.config = config;

    const apiKeys = this.resolveApiKeys(config.api_keys);
    this.pool = buildAgentPoolFromConfig(config.agents, apiKeys);
    this.toolLayer = createToolLayer({
      test_command: config.execution.test_command,
      allowed_tools: config.execution.allowed_tools,
      working_directory: config.execution.working_directory,
    });

    const sessionId = crypto.randomUUID();
    this.state = createSessionState(sessionId);
    this.logger = new SessionLogger(sessionId);
  }

  private resolveApiKeys(apiKeys: Record<string, string>): Record<string, string> {
    const resolved: Record<string, string> = {};
    for (const [key, value] of Object.entries(apiKeys)) {
      if (value.startsWith("${") && value.endsWith("}")) {
        const envName = value.slice(2, -1);
        resolved[key] = process.env[envName] || "";
      } else {
        resolved[key] = value;
      }
    }
    return resolved;
  }

  async handleCommand(input: string): Promise<void> {
    if (!input.startsWith("/")) {
      await this.runTask(input);
      return;
    }

    const parts = input.split(/\s+/);
    const command = parts[0].toLowerCase();

    switch (command) {
      case "/help":
        this.showHelp();
        break;
      case "/agents":
        this.showAgents();
        break;
      case "/votes":
        this.showVotes();
        break;
      case "/reset":
        this.reset();
        break;
      case "/export":
        await this.exportLog();
        break;
      case "/skip":
        if (parts[1] === "vote") {
          this.state.fastMode = true;
          messageStore.push(createMessage("orchestrator", "Fast mode enabled. Voting phases will be skipped."));
        }
        break;
      case "/plans":
        this.showPlans();
        break;
      case "/plan":
        this.showPlanDetail(parts[1]);
        break;
      case "/replay":
        await this.handleReplay(parts[1]);
        break;
      default:
        messageStore.push(createMessage("error", `Unknown command: ${command}. Type /help for available commands.`));
    }
  }

  private showHelp(): void {
    const commands = [
      "/help           Show all commands",
      "/agents         List active agents and their model strings",
      "/plans          List all plans with one-line summaries",
      "/plan <N>       Show full detail for plan number N",
      "/votes          Show full vote log for current session",
      "/replay         List saved sessions",
      "/replay <N>     Load and display a saved session",
      "/skip vote      Skip voting phases (fast mode)",
      "/reset          Clear session state",
      "/export         Save session log to JSON",
    ];

    messageStore.push(createMessage("orchestrator", "Available commands:"));
    for (const cmd of commands) {
      messageStore.push(createMessage("dim", `  ${cmd}`));
    }
  }

  private showAgents(): void {
    const summaries = this.pool.getAgentSummaries();
    messageStore.push(createMessage("orchestrator", `Active agents (${summaries.length}):`));
    for (const agent of summaries) {
      messageStore.push(createMessage("dim", `  ${agent.name} (${agent.provider}: ${agent.model}) — ${agent.specialty}`));
    }
  }

  private showVotes(): void {
    if (this.state.planVotes.length === 0 && this.state.executorVotes.length === 0) {
      messageStore.push(createMessage("orchestrator", "No votes have been cast in this session yet."));
      return;
    }

    if (this.state.planVotes.length > 0) {
      messageStore.push(createMessage("orchestrator", `Plan Votes (${this.state.planVotes.length} votes):`));
      for (const vote of this.state.planVotes) {
        messageStore.push(createMessage("dim", `  ${vote.voter} → plan ${vote.voted_for_plan_id.slice(0, 8)}...`));
      }
    }

    if (this.state.executorVotes.length > 0) {
      messageStore.push(createMessage("orchestrator", `Executor Votes (${this.state.executorVotes.length} votes):`));
      for (const vote of this.state.executorVotes) {
        messageStore.push(createMessage("dim", `  ${vote.voter} → ${vote.nominated_executor}`));
      }
    }
  }

  private showPlans(): void {
    if (this.state.plans.size === 0) {
      messageStore.push(createMessage("orchestrator", "No plans have been generated in this session yet."));
      return;
    }

    const planList = [...this.state.plans.values()];
    messageStore.push(createMessage("orchestrator", `Plans (${planList.length}):`));
    planList.forEach((plan, i) => {
      messageStore.push(createMessage("dim", `  ${i + 1}. [${plan.agent}] ${plan.summary} (confidence: ${Math.round(plan.self_confidence * 100)}%)`));
    });
  }

  private showPlanDetail(indexStr: string): void {
    const planList = [...this.state.plans.values()];
    if (planList.length === 0) {
      messageStore.push(createMessage("orchestrator", "No plans have been generated in this session yet."));
      return;
    }
    const idx = parseInt(indexStr, 10);
    if (isNaN(idx) || idx < 1 || idx > planList.length) {
      messageStore.push(createMessage("error", `Invalid plan number. Use /plans to list plans (1-${planList.length}).`));
      return;
    }
    const plan = planList[idx - 1];
    const stepsStr = plan.steps.map((s) => `  ${s.step}. ${s.action}${s.target_file ? ` (${s.target_file})` : ""}`).join("\n");
    messageStore.push(createMessage("orchestrator", `Plan #${idx} by ${plan.agent}:`));
    messageStore.push(createMessage("dim", `  ID: ${plan.plan_id}`));
    messageStore.push(createMessage("dim", `  Summary: ${plan.summary}`));
    messageStore.push(createMessage("dim", `  Steps:\n${stepsStr}`));
    messageStore.push(createMessage("dim", `  Complexity: ${plan.estimated_complexity}`));
    messageStore.push(createMessage("dim", `  Confidence: ${Math.round(plan.self_confidence * 100)}%`));
    if (plan.risks.length > 0) {
      messageStore.push(createMessage("warning", `  Risks: ${plan.risks.join(", ")}`));
    }
  }

  private reset(): void {
    const sessionId = crypto.randomUUID();
    this.state = createSessionState(sessionId);
    this.logger = new SessionLogger(sessionId);
    this.isRunning = false;
    messageStore.push(createMessage("orchestrator", "Session state cleared."));
  }

  private async exportLog(): Promise<void> {
    await this.logger.save();
    messageStore.push(createMessage("success", `Session log saved to ${this.logger.getLogPath()}`));
  }

  private async handleReplay(indexStr: string | undefined): Promise<void> {
    const sessions = await this.logger.listSessions();
    if (sessions.length === 0) {
      messageStore.push(createMessage("orchestrator", "No saved sessions found."));
      return;
    }

    if (!indexStr) {
      messageStore.push(createMessage("orchestrator", "Available sessions:"));
      sessions.forEach((s, i) => {
        const status = s.completedAt ? "completed" : "incomplete";
        messageStore.push(createMessage("dim", `  ${i + 1}. ${s.sessionId.slice(0, 8)}... (${new Date(s.startedAt).toLocaleString()}, ${status})`));
      });
      messageStore.push(createMessage("dim", "  Use /replay <N> to load a session."));
      return;
    }

    const idx = parseInt(indexStr, 10);
    if (isNaN(idx) || idx < 1 || idx > sessions.length) {
      messageStore.push(createMessage("error", `Invalid session number. Use /replay to list sessions (1-${sessions.length}).`));
      return;
    }

    const target = sessions[idx - 1];
    const replayLogger = new SessionLogger(target.sessionId);
    try {
      await replayLogger.load();
      const log = replayLogger.getLog();
      messageStore.push(createMessage("orchestrator", `Session ${target.sessionId.slice(0, 8)}... (started: ${new Date(target.startedAt).toLocaleString()}):`));
      for (const entry of log.entries) {
        messageStore.push(createMessage("dim", `  [${entry.phase}] ${JSON.stringify(entry).slice(0, 200)}`));
      }
    } catch {
      messageStore.push(createMessage("error", `Failed to load session ${target.sessionId.slice(0, 8)}...`));
    }
  }

  async runTask(input: string): Promise<void> {
    if (this.isRunning) {
      messageStore.push(createMessage("error", "A task is already running. Please wait or use /reset."));
      return;
    }

    this.isRunning = true;

    try {
      appState.setPhase("intake");
      await runPhase1Intake(this.state, input, this.config.execution.working_directory);

      const enabledAgents = this.pool.getEnabledAgents();
      if (enabledAgents.length < 3) {
        messageStore.push(createMessage("error", `Minimum 3 agents required, got ${enabledAgents.length}. Add more agents in config.`));
        this.isRunning = false;
        return;
      }

      appState.setPhase("plan_generation");
      await runPhase2Plan(this.state, enabledAgents, this.config);

      if (!this.state.fastMode) {
        appState.setPhase("plan_vote");
        await runPhase3PlanVote(this.state, enabledAgents, this.config.voting);
        appState.setPhase("compatibility_vote");
        await runPhase4CompatVote(this.state, enabledAgents, this.config.voting);
      } else {
        if (this.state.winningPlan) {
          this.state.selectedExecutor = this.state.winningPlan.agent;
          messageStore.push(createMessage("orchestrator", `Fast mode: ${this.state.selectedExecutor} selected as executor.`));
        }
      }

      appState.setPhase("execution");
      await runPhase5Execute(
        this.state,
        enabledAgents,
        this.toolLayer,
        this.config.voting,
        {
          dissent_acknowledgment: this.config.features.dissent_acknowledgment,
          post_execution_peer_review: this.config.features.post_execution_peer_review,
          max_fix_cycles: this.config.execution.max_fix_cycles,
        }
      );

      if (this.config.features.session_logging) {
        await this.logger.save();
      }
    } catch (error) {
      messageStore.push(createMessage("error", `Task failed: ${error instanceof Error ? error.message : String(error)}`));
    } finally {
      this.isRunning = false;
      appState.setRunning(false);
    }
  }

  getState(): SessionState {
    return this.state;
  }

  getPool(): AgentPool {
    return this.pool;
  }
}