import OpenAI from "openai";
import { LLMProvider, ProviderResponse } from "./base.js";

export class OpenAIProvider implements LLMProvider {
  private client: OpenAI;
  private model: string;

  constructor(apiKey: string, model: string, baseUrl?: string) {
    this.client = new OpenAI({
      apiKey,
      baseURL: baseUrl || "https://api.openai.com/v1",
    });
    this.model = model;
  }

  async generate(prompt: string, systemPrompt?: string): Promise<ProviderResponse> {
    const messages: OpenAI.Chat.ChatCompletionMessageParam[] = [];

    if (systemPrompt) {
      messages.push({ role: "system", content: systemPrompt });
    }

    messages.push({ role: "user", content: prompt });

    const completion = await this.client.chat.completions.create({
      model: this.model,
      messages,
      max_tokens: 4096,
    });

    const content = completion.choices[0]?.message?.content || "";

    return { content, raw: completion };
  }
}