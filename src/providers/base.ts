import { TaskBrief, Plan, PlanVote, ExecutorVote, PeerReview } from "../voting/schemas.js";

export interface ProviderResponse {
  content: string;
  raw: unknown;
}

export interface LLMProvider {
  generate(prompt: string, systemPrompt?: string): Promise<ProviderResponse>;
}

export type ProviderFactory = (apiKey: string, model: string, baseUrl?: string) => LLMProvider;