import { describe, it, expect } from "vitest";
import {
  calculatePlanScore,
  aggregatePlanVotes,
  aggregateExecutorVotes,
  normalizeScores,
  renderScoreBar,
} from "../voting/vote-engine.js";
import { Plan, VotingConfig, PlanVote, ExecutorVote } from "../voting/schemas.js";

const testConfig: VotingConfig = {
  weights: {
    correctness: 0.35,
    safety: 0.25,
    completeness: 0.25,
    simplicity: 0.15,
  },
  confidence_threshold: 6.0,
  tie_margin: 0.5,
  self_nomination_penalty: 0.1,
};

const makePlan = (id: string, agent: string): Plan => ({
  agent,
  plan_id: id,
  summary: `${agent}'s plan`,
  steps: [{ step: 1, action: "Do something" }],
  estimated_complexity: "medium",
  risks: [],
  self_confidence: 0.8,
});

describe("calculatePlanScore", () => {
  it("should correctly calculate weighted score", () => {
    const vote: PlanVote = {
      voter: "Haiku",
      voted_for_plan_id: "plan-1",
      scores: { correctness: 8, safety: 9, simplicity: 7, completeness: 8 },
      reason: "test",
    };
    const score = calculatePlanScore(vote, testConfig.weights);
    expect(score).toBeCloseTo(8.1, 2);
  });

  it("should give more weight to correctness", () => {
    const highCorrectness: PlanVote = {
      voter: "A",
      voted_for_plan_id: "p1",
      scores: { correctness: 10, safety: 1, simplicity: 1, completeness: 1 },
      reason: "",
    };
    const highSafety: PlanVote = {
      voter: "B",
      voted_for_plan_id: "p2",
      scores: { correctness: 1, safety: 10, simplicity: 1, completeness: 1 },
      reason: "",
    };
    const scoreA = calculatePlanScore(highCorrectness, testConfig.weights);
    const scoreB = calculatePlanScore(highSafety, testConfig.weights);
    expect(scoreA).toBeGreaterThan(scoreB);
  });
});

describe("aggregatePlanVotes", () => {
  it("should select the plan with highest average score", () => {
    const plans = [makePlan("p1", "Sonnet"), makePlan("p2", "GPT4o")];
    const votes: PlanVote[] = [
      {
        voter: "Haiku",
        voted_for_plan_id: "p1",
        scores: { correctness: 9, safety: 9, simplicity: 8, completeness: 8 },
        reason: "",
      },
      {
        voter: "Opus",
        voted_for_plan_id: "p2",
        scores: { correctness: 5, safety: 5, simplicity: 5, completeness: 5 },
        reason: "",
      },
    ];

    const result = aggregatePlanVotes(votes, plans, testConfig);
    expect(result.winner.plan_id).toBe("p1");
    expect(result.tie).toBe(false);
  });

  it("should detect a tie within margin", () => {
    const plans = [makePlan("p1", "Sonnet"), makePlan("p2", "GPT4o")];
    const votes: PlanVote[] = [
      {
        voter: "Haiku",
        voted_for_plan_id: "p1",
        scores: { correctness: 8, safety: 8, simplicity: 8, completeness: 8 },
        reason: "",
      },
      {
        voter: "Opus",
        voted_for_plan_id: "p2",
        scores: { correctness: 8, safety: 8, simplicity: 8, completeness: 8 },
        reason: "",
      },
    ];

    const result = aggregatePlanVotes(votes, plans, testConfig);
    expect(result.tie).toBe(true);
  });
});

describe("aggregateExecutorVotes", () => {
  it("should select the executor with highest average score", () => {
    const votes: ExecutorVote[] = [
      {
        voter: "Haiku",
        nominated_executor: "DeepSeek",
        reasoning: "Best fit",
        specialty_match_score: 9,
        context_fit_score: 9,
      },
      {
        voter: "Opus",
        nominated_executor: "Sonnet",
        reasoning: "Good fit",
        specialty_match_score: 5,
        context_fit_score: 5,
      },
    ];

    const result = aggregateExecutorVotes(votes, testConfig);
    expect(result.winner).toBe("DeepSeek");
  });

  it("should penalize self-nomination and allow other to win", () => {
    const votes: ExecutorVote[] = [
      {
        voter: "Sonnet",
        nominated_executor: "Sonnet",
        reasoning: "I can do it",
        specialty_match_score: 10,
        context_fit_score: 10,
      },
      {
        voter: "Haiku",
        nominated_executor: "DeepSeek",
        reasoning: "Better fit",
        specialty_match_score: 10,
        context_fit_score: 10,
      },
    ];

    const result = aggregateExecutorVotes(votes, testConfig);
    const sonnetScore = 10 - 10 * 0.1;
    const deepseekScore = 10;
    expect(sonnetScore).toBeLessThan(deepseekScore);
    expect(result.winner).toBe("DeepSeek");
  });
});

describe("normalizeScores", () => {
  it("should normalize scores to 0-10 range", () => {
    const scores = new Map([
      ["a", 2],
      ["b", 4],
      ["c", 8],
    ]);
    const normalized = normalizeScores(scores);
    expect(normalized.get("a")).toBe(0);
    expect(normalized.get("c")).toBe(10);
  });

  it("should return 5 for all when min equals max", () => {
    const scores = new Map([
      ["a", 5],
      ["b", 5],
    ]);
    const normalized = normalizeScores(scores);
    expect(normalized.get("a")).toBe(5);
    expect(normalized.get("b")).toBe(5);
  });
});

describe("renderScoreBar", () => {
  it("should render correct number of filled blocks", () => {
    const bar = renderScoreBar(5, 10);
    const filled = (bar.match(/█/g) || []).length;
    const empty = (bar.match(/░/g) || []).length;
    expect(filled).toBe(5);
    expect(empty).toBe(5);
  });

  it("should handle 0 score", () => {
    const bar = renderScoreBar(0, 10);
    expect(bar).toBe("░".repeat(10));
  });

  it("should handle 10 score", () => {
    const bar = renderScoreBar(10, 10);
    expect(bar).toBe("█".repeat(10));
  });
});