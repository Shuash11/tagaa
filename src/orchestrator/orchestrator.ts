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
      default:
        messageStore.push(createMessage("error", `Unknown command: ${command}. Type /help for available commands.`));
    }
  }

  private showHelp(): void {
    const commands = [
      "/help           Show all commands",
      "/agents         List active agents and their model strings",
      "/votes          Show full vote log for current session",
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

  async runTask(input: string): Promise<void> {
    if (this.isRunning) {
      messageStore.push(createMessage("error", "A task is already running. Please wait or use /reset."));
      return;
    }

    this.isRunning = true;

    try {
      await runPhase1Intake(this.state, input, this.config.execution.working_directory);

      const enabledAgents = this.pool.getEnabledAgents();
      if (enabledAgents.length < 3) {
        messageStore.push(createMessage("error", `Minimum 3 agents required, got ${enabledAgents.length}. Add more agents in config.`));
        this.isRunning = false;
        return;
      }

      await runPhase2Plan(this.state, enabledAgents);

      if (!this.state.fastMode) {
        await runPhase3PlanVote(this.state, enabledAgents, this.config.voting);
        await runPhase4CompatVote(this.state, enabledAgents, this.config.voting);
      } else {
        if (this.state.winningPlan) {
          this.state.selectedExecutor = this.state.winningPlan.agent;
          messageStore.push(createMessage("orchestrator", `Fast mode: ${this.state.selectedExecutor} selected as executor.`));
        }
      }

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
    }
  }

  getState(): SessionState {
    return this.state;
  }

  getPool(): AgentPool {
    return this.pool;
  }
}