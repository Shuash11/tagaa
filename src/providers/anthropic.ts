import Anthropic from "@anthropic-ai/sdk";
import { LLMProvider, ProviderResponse } from "./base.js";

export class AnthropicProvider implements LLMProvider {
  private client: Anthropic;
  private model: string;

  constructor(apiKey: string, model: string) {
    this.client = new Anthropic({ apiKey });
    this.model = model;
  }

  async generate(prompt: string, systemPrompt?: string): Promise<ProviderResponse> {
    const msg = await this.client.messages.create({
      model: this.model,
      max_tokens: 4096,
      system: systemPrompt,
      messages: [{ role: "user", content: prompt }],
    });

    const content = msg.content
      .filter((block) => block.type === "text")
      .map((block) => (block as { text: string }).text)
      .join("\n");

    return { content, raw: msg };
  }
}