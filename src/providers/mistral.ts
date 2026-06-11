import { Mistral } from "@mistralai/mistralai";
import { LLMProvider, ProviderResponse } from "./base.js";

export class MistralProvider implements LLMProvider {
  private client: Mistral;
  private model: string;

  constructor(apiKey: string, model: string) {
    this.client = new Mistral({ apiKey });
    this.model = model;
  }

  async generate(prompt: string, systemPrompt?: string): Promise<ProviderResponse> {
    const messages: { role: string; content: string }[] = [];

    if (systemPrompt) {
      messages.push({ role: "system", content: systemPrompt });
    }

    messages.push({ role: "user", content: prompt });

    const result = await this.client.chat.complete({
      model: this.model,
      messages: messages as never,
      maxTokens: 4096,
    });

    const content = typeof result.choices?.[0]?.message?.content === "string"
      ? result.choices[0].message.content
      : "";

    return { content, raw: result };
  }
}