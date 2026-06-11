import { ToolLayer } from "./tool-layer.js";
import { registerReadFile } from "./read-file.js";
import { registerWriteFile } from "./write-file.js";
import { registerEditFile } from "./edit-file.js";
import { registerRunCommand } from "./run-command.js";
import { registerRunTests } from "./run-tests.js";
import { registerSearchCodebase } from "./search-codebase.js";

export { ToolLayer } from "./tool-layer.js";
export { ReadFileHandler } from "./read-file.js";
export { WriteFileHandler } from "./write-file.js";
export { EditFileHandler } from "./edit-file.js";
export { RunCommandHandler } from "./run-command.js";
export { RunTestsHandler } from "./run-tests.js";
export { SearchCodebaseHandler } from "./search-codebase.js";

export function createToolLayer(config: {
  test_command: string;
  allowed_tools: string[];
  working_directory: string;
}): ToolLayer {
  const layer = new ToolLayer(config.allowed_tools);
  registerReadFile(layer);
  registerWriteFile(layer);
  registerEditFile(layer);
  registerRunCommand(layer);
  registerRunTests(layer, config.test_command);
  registerSearchCodebase(layer);
  return layer;
}