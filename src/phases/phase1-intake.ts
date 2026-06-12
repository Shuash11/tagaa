import crypto from "crypto";
import * as fs from "fs/promises";
import { SessionState } from "../orchestrator/state.js";
import { TaskBriefSchema } from "../voting/schemas.js";
import { messageStore, createMessage } from "../ui/message-store.js";

export async function runPhase1Intake(
  state: SessionState,
  rawInput: string,
  workingDirectory: string = "."
): Promise<void> {
  state.phase = "intake";

  const taskType = classifyTask(rawInput);
  messageStore.push(createMessage("orchestrator", `Task received → Type: ${taskType}`));

  const referencedFiles = extractFilePaths(rawInput);
  const fileContents: Record<string, string> = {};

  for (const filePath of referencedFiles) {
    try {
      const fullPath = filePath.startsWith("/")
        ? filePath
        : `${workingDirectory}/${filePath}`;
      const content = await fs.readFile(fullPath, "utf-8");
      fileContents[filePath] = content;
      messageStore.push(createMessage("orchestrator", `Loaded: ${filePath} (${content.split("\n").length} lines)`));
    } catch (error) {
      messageStore.push(createMessage("warning", `Could not read file: ${filePath}`));
    }
  }

  state.taskBrief = TaskBriefSchema.parse({
    task_id: crypto.randomUUID(),
    raw_input: rawInput,
    classified_type: taskType,
    attached_files: referencedFiles,
    file_contents: fileContents,
    timestamp: new Date().toISOString(),
  });

  messageStore.push(createMessage("orchestrator", "Briefing all agents..."));
}

const TASK_TYPE_PATTERNS: Record<string, RegExp[]> = {
  code_edit: [/edit\b/i, /change\b/i, /modify\b/i, /update\b/i, /refactor\b/i],
  new_feature: [/add\b.*(feature|function|command|option)/i, /create\b.*(new|feature)/i, /implement\b/i],
  bug_fix: [/bug\b/i, /fix\b/i, /error\b/i, /crash\b/i, /race condition/i, /deadlock/i, /issue\b/i],
  research: [/research\b/i, /find\b/i, /search\b/i, /look\b.*up/i, /investigate\b/i],
  refactor: [/refactor\b/i, /clean\b.*up/i, /restructure\b/i, /optimize\b/i],
  shell_task: [/run\b/i, /execute\b/i, /command\b/i, /shell\b/i, /terminal\b/i],
};

function classifyTask(input: string): string {
  for (const [type, patterns] of Object.entries(TASK_TYPE_PATTERNS)) {
    for (const pattern of patterns) {
      if (pattern.test(input)) return type;
    }
  }
  return "general";
}

function extractFilePaths(input: string): string[] {
  const paths: string[] = [];
  const patterns = [
    // backtick-wrapped paths: `path/to/file.ts`
    /`([^`]+\.[a-zA-Z]{1,6})`/g,
    // relative paths with file extension: ./path/to/file.ts, ../path/file.ts
    /(?<!\w)(\.\/|\.\.\/)?[\w-]+\/[\w.\/-]+\.\w{1,6}(?!\w)/g,
  ];

  for (const pattern of patterns) {
    const matches = input.matchAll(pattern);
    for (const match of matches) {
      const path = match[1] || match[0];
      if (!paths.includes(path)) {
        paths.push(path);
      }
    }
  }

  return paths;
}