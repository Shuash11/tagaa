import { ToolCall, ToolResult } from "../voting/schemas.js";

export interface ToolHandler {
  execute(args: Record<string, unknown>): Promise<ToolResult>;
}

export class ToolLayer {
  private handlers: Map<string, ToolHandler> = new Map();
  private allowedTools: Set<string>;

  constructor(allowedTools: string[]) {
    this.allowedTools = new Set(allowedTools);
  }

  registerTool(name: string, handler: ToolHandler): void {
    this.handlers.set(name, handler);
  }

  async execute(toolCall: ToolCall): Promise<ToolResult> {
    if (!this.allowedTools.has(toolCall.tool)) {
      return {
        success: false,
        error: `Tool "${toolCall.tool}" is not in the allowed tools list`,
      };
    }

    const handler = this.handlers.get(toolCall.tool);
    if (!handler) {
      return {
        success: false,
        error: `Unknown tool: ${toolCall.tool}`,
      };
    }

    try {
      return await handler.execute(toolCall.args);
    } catch (error) {
      return {
        success: false,
        error: `Tool execution error: ${error instanceof Error ? error.message : String(error)}`,
      };
    }
  }
}