import { exec } from "child_process";
import { promisify } from "util";
import { ToolHandler, ToolLayer } from "./tool-layer.js";
import { ToolResult } from "../voting/schemas.js";

const execAsync = promisify(exec);

export class RunTestsHandler implements ToolHandler {
  private testCommand: string;

  constructor(testCommand: string) {
    this.testCommand = testCommand;
  }

  async execute(args: Record<string, unknown>): Promise<ToolResult> {
    const command = (args.command as string) || this.testCommand;
    const cwd = (args.cwd as string) || process.cwd();

    try {
      const { stdout, stderr } = await execAsync(command, {
        cwd,
        timeout: 120000,
        maxBuffer: 10 * 1024 * 1024,
      });

      return {
        success: true,
        output: stdout,
        error: stderr || undefined,
      };
    } catch (error: unknown) {
      const err = error as { stdout?: string; stderr?: string; message?: string };
      return {
        success: false,
        output: err.stdout || undefined,
        error: err.stderr || err.message || String(error),
      };
    }
  }
}

export function registerRunTests(toolLayer: ToolLayer, testCommand: string): void {
  toolLayer.registerTool("run_tests", new RunTestsHandler(testCommand));
}