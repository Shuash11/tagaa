import { LLMProvider, ProviderResponse } from "./base.js";
import { OpenAIProvider } from "./openai.js";

export class CompatOpenAIProvider implements LLMProvider {
  private inner: OpenAIProvider;

  constructor(apiKey: string, model: string, baseUrl: string) {
    this.inner = new OpenAIProvider(apiKey, model, baseUrl);
  }

  async generate(prompt: string, systemPrompt?: string): Promise<ProviderResponse> {
    return this.inner.generate(prompt, systemPrompt);
  }
}