import { Agent } from "../agents/agent.js";
import { SessionState } from "../orchestrator/state.js";
import { VotingConfig } from "../voting/schemas.js";
import { aggregateExecutorVotes, renderScoreBar } from "../voting/vote-engine.js";
import { messageStore, createMessage } from "../ui/message-store.js";

export async function runPhase4CompatVote(
  state: SessionState,
  agents: Agent[],
  config: VotingConfig
): Promise<void> {
  state.phase = "compatibility_vote";

  if (!state.winningPlan) {
    throw new Error("Cannot run compatibility vote: no winning plan selected");
  }

  if (!state.taskBrief) {
    throw new Error("Cannot run compatibility vote: no task brief available");
  }

  messageStore.push(createMessage("orchestrator", "Agents voting on best executor for the plan..."));

  const plans = [...state.plans.values()];
  const agentSummaries = agents.map((a) => ({
    name: a.name,
    specialty: a.specialty,
  }));

  const voteResults = await Promise.allSettled(
    agents.map((agent) =>
      agent.voteOnExecutor(plans, state.winningPlan!.plan_id, state.taskBrief!, agentSummaries)
    )
  );

  const votes: {
    voter: string;
    nominated_executor: string;
    reasoning: string;
    specialty_match_score: number;
    context_fit_score: number;
  }[] = [];

  for (const result of voteResults) {
    if (result.status === "fulfilled") {
      votes.push(result.value);
    }
  }

  const { winner, scores } = aggregateExecutorVotes(votes, config);
  state.selectedExecutor = winner;
  state.executorVotes = votes;

  const uniqueExecutors = [...new Set(votes.map((v) => v.nominated_executor))];

  messageStore.push(createMessage("vote", `Executor Scores:`));
  for (const name of uniqueExecutors) {
    const score = scores.get(name) || 0;
    const isWinner = name === winner;
    const bar = renderScoreBar(score, 12);
    const tag = isWinner ? "  ← selected" : "";
    messageStore.push(createMessage("vote", `  ${name.padEnd(10)} ${bar}  ${score.toFixed(2)}${tag}`));
  }

  messageStore.push(createMessage("orchestrator", `Executor elected: ${winner}`));
}