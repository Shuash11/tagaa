import { Agent } from "./agent.js";
import { AgentConfig, TaskBrief, Plan, PlanVote, ExecutorVote, PeerReview } from "../voting/schemas.js";
import { LLMProvider, ProviderFactory } from "../providers/base.js";
import { AnthropicProvider } from "../providers/anthropic.js";
import { OpenAIProvider } from "../providers/openai.js";
import { GoogleProvider } from "../providers/google.js";
import { MistralProvider } from "../providers/mistral.js";
import { CompatOpenAIProvider } from "../providers/compat-openai.js";

export class AgentPool {
  private agents: Agent[] = [];

  addAgent(agent: Agent): void {
    this.agents.push(agent);
  }

  removeAgent(name: string): void {
    this.agents = this.agents.filter((a) => a.name !== name);
  }

  getAgent(name: string): Agent | undefined {
    return this.agents.find((a) => a.name === name);
  }

  getAllAgents(): Agent[] {
    return [...this.agents];
  }

  getEnabledAgents(): Agent[] {
    return this.agents.filter((a) => a.config.enabled);
  }

  getAgentNames(): string[] {
    return this.agents.filter((a) => a.config.enabled).map((a) => a.name);
  }

  getAgentSummaries(): { name: string; specialty: string; provider: string; model: string }[] {
    return this.agents
      .filter((a) => a.config.enabled)
      .map((a) => ({
        name: a.name,
        specialty: a.specialty,
        provider: a.provider,
        model: a.model,
      }));
  }

  size(): number {
    return this.getEnabledAgents().length;
  }
}

export function createProviderForAgent(
  agentConfig: AgentConfig,
  apiKeys: Record<string, string>
): LLMProvider {
  const apiKey = resolveApiKey(agentConfig.provider, apiKeys);

  switch (agentConfig.provider) {
    case "anthropic":
      return new AnthropicProvider(apiKey, agentConfig.model);
    case "openai":
      return new OpenAIProvider(apiKey, agentConfig.model);
    case "google":
      return new GoogleProvider(apiKey, agentConfig.model);
    case "mistral":
      return new MistralProvider(apiKey, agentConfig.model);
    case "deepseek":
      return new CompatOpenAIProvider(
        apiKey,
        agentConfig.model,
        agentConfig.base_url || "https://api.deepseek.com/v1"
      );
    case "xai":
      return new CompatOpenAIProvider(
        apiKey,
        agentConfig.model,
        agentConfig.base_url || "https://api.x.ai/v1"
      );
    case "nvidia":
      return new CompatOpenAIProvider(
        apiKey,
        agentConfig.model,
        agentConfig.base_url || "https://integrate.api.nvidia.com/v1"
      );
    default:
      throw new Error(`Unknown provider: ${agentConfig.provider}`);
  }
}

function resolveApiKey(
  provider: string,
  apiKeys: Record<string, string>
): string {
  const key = apiKeys[provider];
  if (!key || key.startsWith("${")) {
    const envVar = key?.replace(/\${(.+)}/, "$1") || `${provider.toUpperCase()}_API_KEY`;
    return process.env[envVar] || "";
  }
  return key;
}

export function buildAgentPoolFromConfig(
  agentConfigs: AgentConfig[],
  apiKeys: Record<string, string>
): AgentPool {
  const pool = new AgentPool();

  for (const config of agentConfigs) {
    if (!config.enabled) continue;
    try {
      const provider = createProviderForAgent(config, apiKeys);
      const agent = new Agent(config, provider);
      pool.addAgent(agent);
    } catch (error) {
      console.error(`Failed to create agent ${config.name}:`, error);
    }
  }

  return pool;
}