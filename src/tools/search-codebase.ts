import { exec } from "child_process";
import { promisify } from "util";
import * as fs from "fs/promises";
import * as path from "path";
import { ToolHandler, ToolLayer } from "./tool-layer.js";
import { ToolResult } from "../voting/schemas.js";

const execAsync = promisify(exec);

export class SearchCodebaseHandler implements ToolHandler {
  async execute(args: Record<string, unknown>): Promise<ToolResult> {
    const pattern = args.pattern as string;
    const searchPath = (args.path as string) || process.cwd();
    const filePattern = args.file_pattern as string | undefined;

    if (!pattern) {
      return { success: false, error: "Missing required argument: pattern" };
    }

    try {
      let cmd: string;
      if (filePattern) {
        cmd = `rg -n --include "${filePattern}" "${pattern}" "${searchPath}"`;
      } else {
        cmd = `rg -n "${pattern}" "${searchPath}"`;
      }

      try {
        const { stdout } = await execAsync(cmd, { timeout: 30000 });
        return { success: true, output: stdout || "No matches found." };
      } catch {
        return { success: true, output: "No matches found." };
      }
    } catch (error) {
      return {
        success: false,
        error: `Search failed: ${error instanceof Error ? error.message : String(error)}`,
      };
    }
  }
}

export function registerSearchCodebase(toolLayer: ToolLayer): void {
  toolLayer.registerTool("search_codebase", new SearchCodebaseHandler());
}