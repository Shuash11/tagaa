import * as fs from "fs/promises";
import * as path from "path";
import { ToolHandler, ToolLayer } from "./tool-layer.js";
import { ToolResult } from "../voting/schemas.js";

export class WriteFileHandler implements ToolHandler {
  async execute(args: Record<string, unknown>): Promise<ToolResult> {
    const filePath = args.path as string;
    const content = args.content as string;

    if (!filePath) {
      return { success: false, error: "Missing required argument: path" };
    }
    if (content === undefined) {
      return { success: false, error: "Missing required argument: content" };
    }

    try {
      await fs.mkdir(path.dirname(filePath), { recursive: true });
      await fs.writeFile(filePath, content, "utf-8");
      return { success: true, output: `Written ${content.length} bytes to ${filePath}` };
    } catch (error) {
      return {
        success: false,
        error: `Failed to write file: ${error instanceof Error ? error.message : String(error)}`,
      };
    }
  }
}

export function registerWriteFile(toolLayer: ToolLayer): void {
  toolLayer.registerTool("write_file", new WriteFileHandler());
}