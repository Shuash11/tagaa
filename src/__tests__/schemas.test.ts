import { describe, it, expect } from "vitest";
import {
  TaskBriefSchema,
  PlanSchema,
  PlanVoteSchema,
  ExecutorVoteSchema,
  ConfigSchema,
  PeerReviewSchema,
  VoteScoresSchema,
} from "../voting/schemas.js";

describe("TaskBriefSchema", () => {
  it("should validate a correct task brief", () => {
    const brief = {
      task_id: "550e8400-e29b-41d4-a716-446655440000",
      raw_input: "fix the bug in worker.ts",
      classified_type: "bug_fix",
      attached_files: ["worker.ts"],
      file_contents: { "worker.ts": "console.log('hello');" },
      timestamp: "2024-01-01T00:00:00.000Z",
    };
    const result = TaskBriefSchema.parse(brief);
    expect(result.raw_input).toBe("fix the bug in worker.ts");
    expect(result.classified_type).toBe("bug_fix");
  });

  it("should reject invalid task type", () => {
    const brief = {
      task_id: "550e8400-e29b-41d4-a716-446655440000",
      raw_input: "test",
      classified_type: "invalid_type",
      attached_files: [],
      file_contents: {},
      timestamp: "2024-01-01T00:00:00.000Z",
    };
    expect(() => TaskBriefSchema.parse(brief)).toThrow();
  });
});

describe("PlanSchema", () => {
  it("should validate a correct plan", () => {
    const plan = {
      agent: "Sonnet",
      plan_id: "550e8400-e29b-41d4-a716-446655440001",
      summary: "Fix the bug by adding a mutex lock",
      steps: [
        { step: 1, action: "Identify critical section", target_file: "worker.ts" },
        { step: 2, action: "Import mutex primitive" },
      ],
      estimated_complexity: "medium",
      risks: ["Deadlock if lock order changes"],
      self_confidence: 0.85,
    };
    const result = PlanSchema.parse(plan);
    expect(result.agent).toBe("Sonnet");
    expect(result.steps).toHaveLength(2);
  });

  it("should reject invalid complexity", () => {
    const plan = {
      agent: "Sonnet",
      plan_id: "550e8400-e29b-41d4-a716-446655440001",
      summary: "Test",
      steps: [],
      estimated_complexity: "very_high",
      risks: [],
      self_confidence: 0.5,
    };
    expect(() => PlanSchema.parse(plan)).toThrow();
  });
});

describe("PlanVoteSchema", () => {
  it("should validate a correct vote", () => {
    const vote = {
      voter: "Haiku",
      voted_for_plan_id: "550e8400-e29b-41d4-a716-446655440001",
      scores: { correctness: 8, safety: 9, simplicity: 7, completeness: 8 },
      reason: "Best approach for the problem",
    };
    const result = PlanVoteSchema.parse(vote);
    expect(result.scores.correctness).toBe(8);
  });

  it("should reject scores outside 1-10", () => {
    const vote = {
      voter: "Haiku",
      voted_for_plan_id: "550e8400-e29b-41d4-a716-446655440001",
      scores: { correctness: 11, safety: 9, simplicity: 7, completeness: 8 },
      reason: "Test",
    };
    expect(() => PlanVoteSchema.parse(vote)).toThrow();
  });
});

describe("ExecutorVoteSchema", () => {
  it("should validate a correct executor vote", () => {
    const vote = {
      voter: "Gemini",
      nominated_executor: "DeepSeek",
      reasoning: "Best at low-level concurrency",
      specialty_match_score: 9,
      context_fit_score: 8,
    };
    const result = ExecutorVoteSchema.parse(vote);
    expect(result.nominated_executor).toBe("DeepSeek");
  });
});

describe("PeerReviewSchema", () => {
  it("should validate a correct review", () => {
    const review = {
      reviewer: "Opus",
      verdict: "approved",
      comments: [{ file: "src/worker.ts", line: 52, note: "Looks good" }],
    };
    const result = PeerReviewSchema.parse(review);
    expect(result.verdict).toBe("approved");
  });
});

describe("VoteScoresSchema", () => {
  it("should reject scores less than 1", () => {
    expect(() =>
      VoteScoresSchema.parse({
        correctness: 0,
        safety: 9,
        simplicity: 7,
        completeness: 8,
      })
    ).toThrow();
  });
});

describe("ConfigSchema", () => {
  it("should validate a minimal config with 3 agents", () => {
    const config = {
      agents: [
        {
          name: "Sonnet",
          provider: "anthropic",
          model: "claude-3-5-sonnet-20241022",
          specialty: "Architecture",
          color: "cyan",
          enabled: true,
        },
        {
          name: "Haiku",
          provider: "anthropic",
          model: "claude-3-haiku-20240307",
          specialty: "Speed",
          color: "green",
          enabled: true,
        },
        {
          name: "GPT4o",
          provider: "openai",
          model: "gpt-4o",
          specialty: "Code Gen",
          color: "yellow",
          enabled: true,
        },
      ],
      api_keys: {
        anthropic: "${ANTHROPIC_API_KEY}",
        openai: "${OPENAI_API_KEY}",
      },
      voting: {
        weights: {
          correctness: 0.35,
          safety: 0.25,
          completeness: 0.25,
          simplicity: 0.15,
        },
        confidence_threshold: 6.0,
        tie_margin: 0.5,
        self_nomination_penalty: 0.1,
      },
      execution: {
        max_fix_cycles: 3,
        test_command: "npm test",
        allowed_tools: ["read_file", "write_file"],
        working_directory: ".",
      },
      features: {
        cross_examination_round: false,
        post_execution_peer_review: false,
        dissent_acknowledgment: false,
        session_logging: false,
        fast_mode: false,
      },
      ui: {
        message_width: 80,
        show_timestamps: true,
        show_specialty_tags: true,
        syntax_highlighting: true,
      },
    };
    const result = ConfigSchema.parse(config);
    expect(result.agents).toHaveLength(3);
  });
});