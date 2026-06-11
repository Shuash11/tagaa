import {
  PlanVote,
  ExecutorVote,
  BugFixVote,
  Plan,
  VotingConfig,
  VoteScoresSchema,
} from "./schemas.js";

export function calculatePlanScore(vote: PlanVote, weights: VotingConfig["weights"]): number {
  const { correctness, safety, simplicity, completeness } = vote.scores;
  return (
    correctness * weights.correctness +
    safety * weights.safety +
    completeness * weights.completeness +
    simplicity * weights.simplicity
  );
}

export function aggregatePlanVotes(
  votes: PlanVote[],
  plans: Plan[],
  config: VotingConfig
): { winner: Plan; scores: Map<string, number>; tie: boolean } {
  const planScores = new Map<string, number[]>();

  for (const vote of votes) {
    const score = calculatePlanScore(vote, config.weights);
    if (!planScores.has(vote.voted_for_plan_id)) {
      planScores.set(vote.voted_for_plan_id, []);
    }
    planScores.get(vote.voted_for_plan_id)!.push(score);
  }

  const avgScores = new Map<string, number>();
  for (const [planId, scores] of planScores) {
    const avg = scores.reduce((a, b) => a + b, 0) / scores.length;
    avgScores.set(planId, avg);
  }

  const sortedPlans = [...avgScores.entries()].sort((a, b) => b[1] - a[1]);

  if (sortedPlans.length < 2) {
    const winner = plans.find((p) => p.plan_id === sortedPlans[0][0])!;
    return { winner, scores: avgScores, tie: false };
  }

  const [top1, top2] = sortedPlans;
  const tie = Math.abs(top1[1] - top2[1]) <= config.tie_margin;

  const winner = plans.find((p) => p.plan_id === top1[0])!;
  return { winner, scores: avgScores, tie };
}

export function aggregateExecutorVotes(
  votes: ExecutorVote[],
  config: VotingConfig
): { winner: string; scores: Map<string, number> } {
  const executorScores = new Map<string, number[]>();

  for (const vote of votes) {
    let score =
      vote.specialty_match_score * 0.6 + vote.context_fit_score * 0.4;

    if (vote.voter === vote.nominated_executor) {
      score *= 1 - config.self_nomination_penalty;
    }

    if (!executorScores.has(vote.nominated_executor)) {
      executorScores.set(vote.nominated_executor, []);
    }
    executorScores.get(vote.nominated_executor)!.push(score);
  }

  const avgScores = new Map<string, number>();
  for (const [executor, scores] of executorScores) {
    const avg = scores.reduce((a, b) => a + b, 0) / scores.length;
    avgScores.set(executor, avg);
  }

  const winner = [...avgScores.entries()].sort((a, b) => b[1] - a[1])[0][0];
  return { winner, scores: avgScores };
}

export function aggregateBugFixVotes(
  votes: BugFixVote[],
  config: VotingConfig
): { winner: string; scores: Map<string, number>; tie: boolean } {
  const fixScores = new Map<string, number[]>();

  for (const vote of votes) {
    const score = calculatePlanScore(vote, config.weights);
    if (!fixScores.has(vote.voted_for_fix_id)) {
      fixScores.set(vote.voted_for_fix_id, []);
    }
    fixScores.get(vote.voted_for_fix_id)!.push(score);
  }

  const avgScores = new Map<string, number>();
  for (const [fixId, scores] of fixScores) {
    const avg = scores.reduce((a, b) => a + b, 0) / scores.length;
    avgScores.set(fixId, avg);
  }

  const sortedFixes = [...avgScores.entries()].sort((a, b) => b[1] - a[1]);

  if (sortedFixes.length < 2) {
    return { winner: sortedFixes[0][0], scores: avgScores, tie: false };
  }

  const [top1, top2] = sortedFixes;
  const tie = Math.abs(top1[1] - top2[1]) <= config.tie_margin;

  return { winner: top1[0], scores: avgScores, tie };
}

export function normalizeScores(
  scores: Map<string, number>
): Map<string, number> {
  const values = [...scores.values()];
  const min = Math.min(...values);
  const max = Math.max(...values);

  if (max === min) {
    const normalized = new Map<string, number>();
    for (const [key] of scores) {
      normalized.set(key, 5);
    }
    return normalized;
  }

  const normalized = new Map<string, number>();
  for (const [key, value] of scores) {
    normalized.set(key, ((value - min) / (max - min)) * 10);
  }
  return normalized;
}

export function renderScoreBar(score: number, maxWidth: number = 20): string {
  const filled = Math.round((score / 10) * maxWidth);
  const empty = maxWidth - filled;
  return "█".repeat(filled) + "░".repeat(empty);
}