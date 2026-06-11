import * as fs from "fs/promises";
import { ToolHandler, ToolLayer } from "./tool-layer.js";
import { ToolResult } from "../voting/schemas.js";

export class EditFileHandler implements ToolHandler {
  async execute(args: Record<string, unknown>): Promise<ToolResult> {
    const filePath = args.path as string;
    const oldContent = args.old_content as string;
    const newContent = args.new_content as string;

    if (!filePath) {
      return { success: false, error: "Missing required argument: path" };
    }
    if (oldContent === undefined && newContent === undefined) {
      return { success: false, error: "Missing required arguments: old_content or new_content" };
    }

    try {
      const currentContent = await fs.readFile(filePath, "utf-8");

      if (oldContent !== undefined) {
        if (!currentContent.includes(oldContent)) {
          return {
            success: false,
            error: `Could not find old_content in ${filePath}`,
          };
        }
        const updated = currentContent.replace(oldContent, newContent);
        await fs.writeFile(filePath, updated, "utf-8");
        return { success: true, output: `Edited ${filePath} (replaced 1 occurrence)` };
      }

      await fs.writeFile(filePath, newContent, "utf-8");
      return { success: true, output: `Written ${newContent.length} bytes to ${filePath}` };
    } catch (error) {
      return {
        success: false,
        error: `Failed to edit file: ${error instanceof Error ? error.message : String(error)}`,
      };
    }
  }
}

export function registerEditFile(toolLayer: ToolLayer): void {
  toolLayer.registerTool("edit_file", new EditFileHandler());
}