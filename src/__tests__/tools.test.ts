import { describe, it, expect, beforeAll, afterAll } from "vitest";
import * as fs from "fs/promises";
import * as path from "path";
import { ToolLayer } from "../tools/tool-layer.js";
import { createToolLayer } from "../tools/index.js";

const testDir = path.join(process.cwd(), "src/__tests__/temp");

describe("ToolLayer", () => {
  beforeAll(async () => {
    await fs.mkdir(testDir, { recursive: true });
    await fs.writeFile(path.join(testDir, "test.txt"), "Hello, World!", "utf-8");
  });

  afterAll(async () => {
    await fs.rm(testDir, { recursive: true, force: true });
  });

  it("should reject unknown tools", async () => {
    const layer = createToolLayer({
      test_command: "echo test",
      allowed_tools: ["read_file"],
      working_directory: ".",
    });

    const result = await layer.execute({ tool: "read_file" as never, args: {} });
    expect(result.success).toBe(false);
  });

  it("should reject disallowed tools", async () => {
    const layer = createToolLayer({
      test_command: "echo test",
      allowed_tools: ["read_file"],
      working_directory: ".",
    });

    const result = await layer.execute({
      tool: "run_command",
      args: { command: "echo hi" },
    });
    expect(result.success).toBe(false);
  });
});