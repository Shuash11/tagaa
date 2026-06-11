#!/usr/bin/env node

import * as fs from "fs/promises";
import * as path from "path";
import readline from "readline";
import React from "react";
import { render } from "ink";
import { Config, ConfigSchema } from "./voting/schemas.js";
import { Orchestrator } from "./orchestrator/orchestrator.js";
import { ChatApp } from "./ui/chat-app.js";
import { messageStore, createMessage } from "./ui/message-store.js";

const CONFIG_PATH = path.join(process.cwd(), "tagaa.config.json");

async function loadConfig(configPath: string): Promise<Config> {
  try {
    const raw = await fs.readFile(configPath, "utf-8");
    const parsed = JSON.parse(raw);
    return ConfigSchema.parse(parsed);
  } catch (error) {
    if (error instanceof Error) {
      throw new Error(`Failed to load config: ${error.message}`);
    }
    throw error;
  }
}

function createDefaultConfig(): Config {
  return ConfigSchema.parse({
    agents: [
      { name: "Sonnet", provider: "anthropic", model: "claude-3-5-sonnet-20241022", specialty: "Architecture, Planning", color: "cyan", enabled: true },
      { name: "Haiku", provider: "anthropic", model: "claude-3-haiku-20240307", specialty: "Speed, Summarization", color: "green", enabled: true },
      { name: "GPT4o", provider: "openai", model: "gpt-4o", specialty: "General, Code Gen", color: "yellow", enabled: true },
    ],
    api_keys: {
      anthropic: "${ANTHROPIC_API_KEY}",
      openai: "${OPENAI_API_KEY}",
      google: "${GOOGLE_API_KEY}",
      mistral: "${MISTRAL_API_KEY}",
      deepseek: "${DEEPSEEK_API_KEY}",
      xai: "${XAI_API_KEY}",
      nvidia: "${NVIDIA_API_KEY}",
    },
    voting: {
      weights: { correctness: 0.35, safety: 0.25, completeness: 0.25, simplicity: 0.15 },
      confidence_threshold: 6.0,
      tie_margin: 0.5,
      self_nomination_penalty: 0.1,
    },
    execution: {
      max_fix_cycles: 3,
      test_command: "npm test",
      allowed_tools: ["read_file", "write_file", "edit_file", "run_command", "run_tests", "search_codebase", "fetch_url"],
      working_directory: ".",
    },
    features: {
      cross_examination_round: true,
      post_execution_peer_review: true,
      dissent_acknowledgment: true,
      session_logging: true,
      fast_mode: false,
    },
    ui: {
      message_width: 80,
      show_timestamps: true,
      show_specialty_tags: true,
      syntax_highlighting: true,
    },
  });
}

function runReadlineInterface(orchestrator: Orchestrator): void {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  console.log("[TAGAA] Type a task or /help");

  rl.on("line", async (line: string) => {
    const input = line.trim();
    if (!input) {
      console.log("\n[TAGAA] Type a task or /help");
      return;
    }
    console.log(`[You] ${input}`);
    await orchestrator.handleCommand(input);
    console.log("\n[TAGAA] Type a task or /help");
  });

  rl.on("close", () => {
    console.log("\n[TAGAA] Session ended.");
    process.exit(0);
  });
}

async function main(): Promise<void> {
  messageStore.clear();

  let config: Config;
  try {
    config = await loadConfig(CONFIG_PATH);
  } catch {
    messageStore.push(createMessage("warning", "No config found. Creating default tagaa.config.json..."));
    config = createDefaultConfig();
    await fs.writeFile(CONFIG_PATH, JSON.stringify(config, null, 2), "utf-8");
    messageStore.push(createMessage("success", `Default config written to ${CONFIG_PATH}`));
  }

  const orchestrator = new Orchestrator(config);
  const enabledAgents = orchestrator.getPool().getEnabledAgents();

  if (!process.stdin.isTTY) {
    runReadlineInterface(orchestrator);
    return;
  }

  messageStore.disableConsoleLog();

  messageStore.push(createMessage("orchestrator", `TAGAA v0.1.0 — Terminal Autonomous Group AI Assistant`));
  messageStore.push(createMessage("orchestrator", `${enabledAgents.length} agents loaded. Type a task or /help`));
  messageStore.push(createMessage("dim", "Ctrl+B toggle sidebar · Ctrl+S configure API keys"));
  messageStore.push(createMessage("divider", ""));

  const agentInfo = enabledAgents.map((a) => ({
    name: a.name,
    provider: a.provider,
    model: a.model,
    specialty: a.specialty,
    color: a.config.color || "white",
    enabled: a.config.enabled,
  }));

  const { waitUntilExit } = render(
    React.createElement(ChatApp, {
      agents: agentInfo,
      onInput: async (input: string) => {
        messageStore.push(createMessage("dim", `You: ${input}`));
        await orchestrator.handleCommand(input);
        messageStore.push(createMessage("divider", ""));
      },
    })
  );

  process.stdin.on("end", () => {
    process.exit(0);
  });

  await waitUntilExit();
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});

export { Orchestrator, loadConfig };
export type { Config } from "./voting/schemas.js";
export { createSessionState } from "./orchestrator/state.js";
export { AgentPool, buildAgentPoolFromConfig, createProviderForAgent } from "./agents/pool.js";
export { Agent } from "./agents/agent.js";
export { createToolLayer } from "./tools/index.js";
export { SessionLogger } from "./logger/session-logger.js";
export { BugFixCouncil } from "./bugfix/council.js";
export {
  calculatePlanScore,
  aggregatePlanVotes,
  aggregateExecutorVotes,
  normalizeScores,
  renderScoreBar,
} from "./voting/vote-engine.js";
export * from "./voting/schemas.js";