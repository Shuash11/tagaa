import { Agent } from "../agents/agent.js";
import { SessionState } from "../orchestrator/state.js";
import { VotingConfig } from "../voting/schemas.js";
import { aggregatePlanVotes, renderScoreBar } from "../voting/vote-engine.js";
import { messageStore, createMessage } from "../ui/message-store.js";

export async function runPhase3PlanVote(
  state: SessionState,
  agents: Agent[],
  config: VotingConfig
): Promise<void> {
  state.phase = "plan_vote";

  const plans = [...state.plans.values()];
  if (plans.length < 2) {
    messageStore.push(createMessage("orchestrator", "Only one plan. Skipping vote."));
    state.winningPlan = plans[0];
    return;
  }

  messageStore.push(createMessage("orchestrator", "Agents are reviewing and scoring each other's plans..."));

  if (!state.taskBrief) {
    throw new Error("Cannot run plan vote: no task brief available");
  }

  const voteResults = await Promise.allSettled(
    agents.map((agent) => agent.voteOnPlans(plans, state.taskBrief!))
  );

  const votes: {
    voter: string;
    voted_for_plan_id: string;
    scores: { correctness: number; safety: number; simplicity: number; completeness: number };
    reason: string;
  }[] = [];

  for (const result of voteResults) {
    if (result.status === "fulfilled") {
      votes.push(result.value);
    }
  }

  const { winner, scores, tie } = aggregatePlanVotes(
    votes.map((v) => ({
      voter: v.voter,
      voted_for_plan_id: v.voted_for_plan_id,
      scores: v.scores,
      reason: v.reason,
    })),
    plans,
    config
  );

  state.winningPlan = winner;
  state.planVotes = votes;

  messageStore.push(createMessage("vote", `Plan Scores:`));
  for (const plan of plans) {
    const score = scores.get(plan.plan_id) || 0;
    const isWinner = plan.plan_id === winner.plan_id;
    const bar = renderScoreBar(score, 12);
    const tag = isWinner ? "  ← winner" : "";
    messageStore.push(createMessage("vote", `  ${plan.agent.padEnd(10)} ${bar}  ${score.toFixed(2)}${tag}`));
  }

  if (tie) messageStore.push(createMessage("warning", "Tie detected — tiebreak round needed."));

  const dissentingVotes = votes.filter((v) => v.voted_for_plan_id !== winner.plan_id);
  state.dissentVotes = dissentingVotes.map((v) => ({
    voter: v.voter,
    voted_for_plan_id: v.voted_for_plan_id,
    scores: v.scores,
    reason: v.reason,
  }));

  for (const dissent of dissentingVotes) {
    const agent = agents.find((a) => a.name === dissent.voter);
    messageStore.push(createMessage("agent", `Dissenting: ${dissent.reason}`, {
      agentName: dissent.voter,
      color: agent?.color || "white",
    }));
  }

  messageStore.push(createMessage("orchestrator", `Winner: ${winner.agent}'s plan (score: ${(scores.get(winner.plan_id) || 0).toFixed(2)})`));
}