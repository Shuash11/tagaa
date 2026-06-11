import * as fs from "fs/promises";
import { ToolHandler, ToolLayer } from "./tool-layer.js";
import { ToolResult } from "../voting/schemas.js";

export class ReadFileHandler implements ToolHandler {
  async execute(args: Record<string, unknown>): Promise<ToolResult> {
    const path = args.path as string;
    if (!path) {
      return { success: false, error: "Missing required argument: path" };
    }

    try {
      const content = await fs.readFile(path, "utf-8");
      return { success: true, output: content };
    } catch (error) {
      return {
        success: false,
        error: `Failed to read file: ${error instanceof Error ? error.message : String(error)}`,
      };
    }
  }
}

export function registerReadFile(toolLayer: ToolLayer): void {
  toolLayer.registerTool("read_file", new ReadFileHandler());
}