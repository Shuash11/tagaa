import { z } from "zod";

export const TaskTypeSchema = z.enum([
  "code_edit",
  "new_feature",
  "bug_fix",
  "research",
  "refactor",
  "shell_task",
  "general",
]);

export const TaskBriefSchema = z.object({
  task_id: z.string().uuid(),
  raw_input: z.string(),
  classified_type: TaskTypeSchema,
  attached_files: z.array(z.string()),
  file_contents: z.record(z.string()),
  timestamp: z.string().datetime(),
});

export type TaskBrief = z.infer<typeof TaskBriefSchema>;

export const PlanStepSchema = z.object({
  step: z.number().int().positive(),
  action: z.string(),
  target_file: z.string().optional(),
});

export const PlanSchema = z.object({
  agent: z.string(),
  plan_id: z.string().uuid(),
  summary: z.string(),
  steps: z.array(PlanStepSchema),
  estimated_complexity: z.enum(["low", "medium", "high"]),
  risks: z.array(z.string()),
  self_confidence: z.number().min(0).max(1),
});

export type Plan = z.infer<typeof PlanSchema>;

export const VoteScoresSchema = z.object({
  correctness: z.number().int().min(1).max(10),
  safety: z.number().int().min(1).max(10),
  simplicity: z.number().int().min(1).max(10),
  completeness: z.number().int().min(1).max(10),
});

export const PlanVoteSchema = z.object({
  voter: z.string(),
  voted_for_plan_id: z.string().uuid(),
  scores: VoteScoresSchema,
  reason: z.string(),
});

export type PlanVote = z.infer<typeof PlanVoteSchema>;

export const ExecutorVoteSchema = z.object({
  voter: z.string(),
  nominated_executor: z.string(),
  reasoning: z.string(),
  specialty_match_score: z.number().int().min(1).max(10),
  context_fit_score: z.number().int().min(1).max(10),
});

export type ExecutorVote = z.infer<typeof ExecutorVoteSchema>;

export const BugFixPlanSchema = PlanSchema.extend({
  original_error: z.string(),
});

export type BugFixPlan = z.infer<typeof BugFixPlanSchema>;

export const BugFixVoteSchema = PlanVoteSchema.extend({
  voted_for_fix_id: z.string().uuid(),
});

export type BugFixVote = z.infer<typeof BugFixVoteSchema>;

export const ToolCallSchema = z.object({
  tool: z.enum([
    "read_file",
    "write_file",
    "edit_file",
    "run_command",
    "run_tests",
    "search_codebase",
    "fetch_url",
  ]),
  args: z.record(z.unknown()),
});

export type ToolCall = z.infer<typeof ToolCallSchema>;

export const ToolResultSchema = z.object({
  success: z.boolean(),
  output: z.string().optional(),
  error: z.string().optional(),
});

export type ToolResult = z.infer<typeof ToolResultSchema>;

export const ExecutionOutputSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("file_edit"),
    path: z.string(),
    diff: z.string(),
  }),
  z.object({
    type: z.literal("bash_cmd"),
    command: z.string(),
    stdout: z.string(),
    stderr: z.string(),
    exit_code: z.number().int(),
  }),
  z.object({
    type: z.literal("code_block"),
    language: z.string(),
    code: z.string(),
  }),
  z.object({
    type: z.literal("message"),
    content: z.string(),
  }),
]);

export type ExecutionOutput = z.infer<typeof ExecutionOutputSchema>;

export const ReviewCommentSchema = z.object({
  file: z.string(),
  line: z.number().int().positive(),
  note: z.string(),
});

export const PeerReviewSchema = z.object({
  reviewer: z.string(),
  verdict: z.enum(["approved", "changes_requested"]),
  comments: z.array(ReviewCommentSchema),
});

export type PeerReview = z.infer<typeof PeerReviewSchema>;

export const SessionLogEntrySchema = z.discriminatedUnion("phase", [
  z.object({
    phase: z.literal("plan_vote"),
    round: z.number().int().positive(),
    votes: z.array(PlanVoteSchema),
    result: z.object({
      winner_plan_id: z.string().uuid(),
      final_score: z.number(),
    }),
  }),
  z.object({
    phase: z.literal("executor_vote"),
    round: z.number().int().positive(),
    votes: z.array(ExecutorVoteSchema),
    result: z.object({
      winner_executor: z.string(),
      final_score: z.number(),
    }),
  }),
  z.object({
    phase: z.literal("bug_fix_vote"),
    round: z.number().int().positive(),
    votes: z.array(BugFixVoteSchema),
    result: z.object({
      winner_fix_id: z.string().uuid(),
      final_score: z.number(),
    }),
  }),
  z.object({
    phase: z.literal("execution"),
    outputs: z.array(ExecutionOutputSchema),
  }),
  z.object({
    phase: z.literal("peer_review"),
    reviews: z.array(PeerReviewSchema),
  }),
]);

export type SessionLogEntry = z.infer<typeof SessionLogEntrySchema>;

export const AgentConfigSchema = z.object({
  name: z.string(),
  provider: z.enum(["anthropic", "openai", "google", "mistral", "deepseek", "xai", "nvidia"]),
  model: z.string(),
  specialty: z.string(),
  color: z.string(),
  enabled: z.boolean(),
  base_url: z.string().url().optional(),
});

export type AgentConfig = z.infer<typeof AgentConfigSchema>;

export const VotingConfigSchema = z.object({
  weights: z.object({
    correctness: z.number().min(0).max(1),
    safety: z.number().min(0).max(1),
    completeness: z.number().min(0).max(1),
    simplicity: z.number().min(0).max(1),
  }),
  confidence_threshold: z.number().min(0).max(10),
  tie_margin: z.number().min(0).max(1),
  self_nomination_penalty: z.number().min(0).max(1),
});

export type VotingConfig = z.infer<typeof VotingConfigSchema>;

export const ExecutionConfigSchema = z.object({
  max_fix_cycles: z.number().int().positive(),
  test_command: z.string(),
  allowed_tools: z.array(z.string()),
  working_directory: z.string(),
});

export type ExecutionConfig = z.infer<typeof ExecutionConfigSchema>;

export const FeaturesConfigSchema = z.object({
  cross_examination_round: z.boolean(),
  post_execution_peer_review: z.boolean(),
  dissent_acknowledgment: z.boolean(),
  session_logging: z.boolean(),
  fast_mode: z.boolean(),
});

export type FeaturesConfig = z.infer<typeof FeaturesConfigSchema>;

export const UIConfigSchema = z.object({
  message_width: z.number().int().positive(),
  show_timestamps: z.boolean(),
  show_specialty_tags: z.boolean(),
  syntax_highlighting: z.boolean(),
});

export type UIConfig = z.infer<typeof UIConfigSchema>;

export const ConfigSchema = z.object({
  agents: z.array(AgentConfigSchema),
  api_keys: z.object({
    anthropic: z.string().optional(),
    openai: z.string().optional(),
    google: z.string().optional(),
    mistral: z.string().optional(),
    deepseek: z.string().optional(),
    xai: z.string().optional(),
    nvidia: z.string().optional(),
  }),
  voting: VotingConfigSchema,
  execution: ExecutionConfigSchema,
  features: FeaturesConfigSchema,
  ui: UIConfigSchema,
});

export type Config = z.infer<typeof ConfigSchema>;