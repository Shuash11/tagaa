import { GoogleGenAI } from "@google/genai";
import { LLMProvider, ProviderResponse } from "./base.js";

export class GoogleProvider implements LLMProvider {
  private client: GoogleGenAI;
  private model: string;

  constructor(apiKey: string, model: string) {
    this.client = new GoogleGenAI({ apiKey });
    this.model = model;
  }

  async generate(prompt: string, systemPrompt?: string): Promise<ProviderResponse> {
    const result = await this.client.models.generateContent({
      model: this.model,
      contents: prompt,
      config: {
        systemInstruction: systemPrompt,
        maxOutputTokens: 4096,
      },
    });

    const content = result.text || "";

    return { content, raw: result };
  }
}