import { describe, it, expect, beforeAll } from "vitest";
import { createSessionState } from "../orchestrator/state.js";
import { aggregatePlanVotes, aggregateExecutorVotes, calculatePlanScore } from "../voting/vote-engine.js";
import { Plan, VotingConfig, PlanVote, ExecutorVote, TaskBrief } from "../voting/schemas.js";

const testConfig: VotingConfig = {
  weights: { correctness: 0.35, safety: 0.25, completeness: 0.25, simplicity: 0.15 },
  confidence_threshold: 6.0,
  tie_margin: 0.5,
  self_nomination_penalty: 0.1,
};

describe("End-to-end workflow simulation", () => {
  it("should simulate the full 5-phase workflow", () => {
    const state = createSessionState("e2e-test");
    expect(state.phase).toBe("standby");

    // Phase 1: Intake
    const taskBrief: TaskBrief = {
      task_id: "550e8400-e29b-41d4-a716-446655440000",
      raw_input: "fix the race condition in worker.ts",
      classified_type: "bug_fix",
      attached_files: ["worker.ts"],
      file_contents: { "worker.ts": "function worker() { /* shared state */ }" },
      timestamp: "2024-01-01T00:00:00.000Z",
    };
    state.taskBrief = taskBrief;
    state.phase = "intake";
    expect(state.taskBrief.classified_type).toBe("bug_fix");
    expect(state.taskBrief.attached_files).toContain("worker.ts");

    // Phase 2: Plan generation
    state.phase = "plan_generation";
    const plans: Plan[] = [
      {
        agent: "Sonnet",
        plan_id: "p1",
        summary: "Wrap shared queue access with a mutex lock",
        steps: [
          { step: 1, action: "Identify critical section", target_file: "worker.ts" },
          { step: 2, action: "Import mutex primitive" },
          { step: 3, action: "Wrap enqueue/dequeue calls" },
        ],
        estimated_complexity: "medium",
        risks: ["Deadlock if lock order changes"],
        self_confidence: 0.9,
      },
      {
        agent: "GPT4o",
        plan_id: "p2",
        summary: "Replace shared queue with lock-free MPSC channel",
        steps: [
          { step: 1, action: "Audit current data flow" },
          { step: 2, action: "Swap queue implementation" },
        ],
        estimated_complexity: "high",
        risks: ["New concurrency primitives needed"],
        self_confidence: 0.7,
      },
      {
        agent: "DeepSeek",
        plan_id: "p3",
        summary: "Use a semaphore with a bounded counter",
        steps: [{ step: 1, action: "Add semaphore to shared resource" }],
        estimated_complexity: "low",
        risks: ["Might not handle all edge cases"],
        self_confidence: 0.6,
      },
    ];

    for (const plan of plans) {
      state.plans.set(plan.plan_id, plan);
    }

    expect(state.plans.size).toBe(3);

    // Phase 3: Plan vote
    state.phase = "plan_vote";
    const planVotes: PlanVote[] = [
      {
        voter: "Haiku",
        voted_for_plan_id: "p1",
        scores: { correctness: 8, safety: 9, simplicity: 7, completeness: 8 },
        reason: "Mutex approach is battle-tested",
      },
      {
        voter: "Opus",
        voted_for_plan_id: "p1",
        scores: { correctness: 9, safety: 8, simplicity: 6, completeness: 9 },
        reason: "Most conservative fix",
      },
      {
        voter: "GPT4o",
        voted_for_plan_id: "p2",
        scores: { correctness: 7, safety: 6, simplicity: 8, completeness: 7 },
        reason: "More elegant long-term",
      },
    ];

    const planVoteResult = aggregatePlanVotes(planVotes, plans, testConfig);
    state.winningPlan = planVoteResult.winner;
    state.planVotes = planVotes;

    expect(planVoteResult.winner.plan_id).toBe("p1");
    expect(state.winningPlan.agent).toBe("Sonnet");
    expect(planVoteResult.scores.get("p1")).toBeGreaterThan(
      planVoteResult.scores.get("p2")!
    );

    // Check confidence threshold
    const winnerScore = planVoteResult.scores.get("p1")!;
    expect(winnerScore).toBeGreaterThanOrEqual(testConfig.confidence_threshold);

    // Phase 4: Compatibility vote
    state.phase = "compatibility_vote";
    const executorVotes: ExecutorVote[] = [
      {
        voter: "Haiku",
        nominated_executor: "DeepSeek",
        reasoning: "Strong at low-level concurrency",
        specialty_match_score: 9,
        context_fit_score: 8,
      },
      {
        voter: "Opus",
        nominated_executor: "Sonnet",
        reasoning: "Knows the plan best",
        specialty_match_score: 7,
        context_fit_score: 9,
      },
      {
        voter: "GPT4o",
        nominated_executor: "DeepSeek",
        reasoning: "DeepSeek is strongest for mutex patterns",
        specialty_match_score: 9,
        context_fit_score: 8,
      },
    ];

    const executorResult = aggregateExecutorVotes(executorVotes, testConfig);
    state.selectedExecutor = executorResult.winner;
    state.executorVotes = executorVotes;

    expect(state.selectedExecutor).toBe("DeepSeek");

    // Phase 5: Execution
    state.phase = "execution";
    state.executionOutputs = [
      {
        type: "bash_cmd",
        command: "echo 'Added mutex lock'",
        stdout: "Added mutex lock",
        stderr: "",
        exit_code: 0,
      },
    ];

    expect(state.executionOutputs).toHaveLength(1);
    const out = state.executionOutputs[0];
    if (out.type === "bash_cmd") {
      expect(out.exit_code).toBe(0);
    }

    // Complete
    state.phase = "complete";
    state.completedAt = new Date().toISOString();
    expect(state.phase).toBe("complete");
    expect(state.completedAt).toBeTruthy();
  });
});