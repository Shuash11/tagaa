import { Agent } from "../agents/agent.js";
import { Plan, TaskBrief } from "../voting/schemas.js";
import { aggregatePlanVotes, aggregateExecutorVotes } from "../voting/vote-engine.js";
import { VotingConfig } from "../voting/schemas.js";
import { SessionState } from "../orchestrator/state.js";

export interface BugFixResult {
  fixed: boolean;
  fixPlan: Plan | null;
  executor: string | null;
  cycle: number;
  error: string | null;
}

export class BugFixCouncil {
  private agents: Agent[];
  private config: VotingConfig;
  private state: SessionState;
  private maxCycles: number;

  constructor(
    agents: Agent[],
    config: VotingConfig,
    state: SessionState,
    maxCycles: number = 3
  ) {
    this.agents = agents;
    this.config = config;
    this.state = state;
    this.maxCycles = maxCycles;
  }

  async run(errorOutput: string, taskBrief: TaskBrief, originalPlan: Plan): Promise<BugFixResult> {
    for (let cycle = 1; cycle <= this.maxCycles; cycle++) {
      this.state.bugFixCycles = cycle;

      const fixPlans = await this.generateFixPlans(errorOutput, taskBrief, originalPlan);
      if (fixPlans.length === 0) {
        return {
          fixed: false,
          fixPlan: null,
          executor: null,
          cycle,
          error: "No fix plans generated",
        };
      }

      const votes = await this.collectFixVotes(fixPlans, taskBrief);

      const { winner, scores } = aggregatePlanVotes(
        votes.map((v) => ({
          voter: v.voter,
          voted_for_plan_id: v.voted_for_plan_id,
          scores: v.scores,
          reason: v.reason,
        })),
        fixPlans,
        this.config
      );

      if (winner && scores.get(winner.plan_id)! >= this.config.confidence_threshold) {
        const executorVotes = await this.collectExecutorVotes(fixPlans, winner, taskBrief);
        const executorResult = aggregateExecutorVotes(executorVotes, this.config);

        return {
          fixed: false,
          fixPlan: winner,
          executor: executorResult.winner,
          cycle,
          error: null,
        };
      }

      if (cycle === this.maxCycles) {
        return {
          fixed: false,
          fixPlan: winner || null,
          executor: null,
          cycle,
          error: "Max fix cycles reached without resolution",
        };
      }
    }

    return {
      fixed: false,
      fixPlan: null,
      executor: null,
      cycle: this.maxCycles,
      error: "Bug Fix Council exhausted",
    };
  }

  private async generateFixPlans(
    errorOutput: string,
    taskBrief: TaskBrief,
    originalPlan: Plan
  ): Promise<Plan[]> {
    const plans: Plan[] = [];
    for (const agent of this.agents) {
      try {
        const plan = await agent.proposeBugFix(errorOutput, taskBrief, originalPlan);
        plans.push(plan);
      } catch (error) {
        console.error(`Agent ${agent.name} failed to propose bug fix:`, error);
      }
    }
    return plans;
  }

  private async collectFixVotes(
    fixPlans: Plan[],
    taskBrief: TaskBrief
  ): Promise<
    { voter: string; voted_for_plan_id: string; scores: { correctness: number; safety: number; simplicity: number; completeness: number }; reason: string }[]
  > {
    const votes: {
      voter: string;
      voted_for_plan_id: string;
      scores: { correctness: number; safety: number; simplicity: number; completeness: number };
      reason: string;
    }[] = [];

    for (const agent of this.agents) {
      try {
        const vote = await agent.voteOnPlans(fixPlans, taskBrief);
        votes.push({
          voter: vote.voter,
          voted_for_plan_id: vote.voted_for_plan_id,
          scores: vote.scores,
          reason: vote.reason,
        });
      } catch (error) {
        console.error(`Agent ${agent.name} failed to vote on fix:`, error);
      }
    }

    return votes;
  }

  private async collectExecutorVotes(
    fixPlans: Plan[],
    winningPlan: Plan,
    taskBrief: TaskBrief
  ): Promise<
    {
      voter: string;
      nominated_executor: string;
      reasoning: string;
      specialty_match_score: number;
      context_fit_score: number;
    }[]
  > {
    const agentSummaries = this.agents.map((a) => ({
      name: a.name,
      specialty: a.specialty,
    }));

    const votes: {
      voter: string;
      nominated_executor: string;
      reasoning: string;
      specialty_match_score: number;
      context_fit_score: number;
    }[] = [];

    for (const agent of this.agents) {
      try {
        const vote = await agent.voteOnExecutor(
          fixPlans,
          winningPlan.plan_id,
          taskBrief,
          agentSummaries
        );
        votes.push(vote);
      } catch (error) {
        console.error(`Agent ${agent.name} failed to vote on fix executor:`, error);
      }
    }

    return votes;
  }
}