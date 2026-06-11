import { TaskBrief, Plan, PlanVote, ExecutorVote, PeerReview, ExecutionOutput } from "../voting/schemas.js";

export type Phase =
  | "standby"
  | "intake"
  | "plan_generation"
  | "plan_vote"
  | "compatibility_vote"
  | "execution"
  | "bug_fix_council"
  | "peer_review"
  | "complete";

export interface SessionState {
  sessionId: string;
  phase: Phase;
  taskBrief: TaskBrief | null;
  plans: Map<string, Plan>;
  planVotes: PlanVote[];
  executorVotes: ExecutorVote[];
  winningPlan: Plan | null;
  selectedExecutor: string | null;
  executionOutputs: ExecutionOutput[];
  bugFixCycles: number;
  peerReviews: PeerReview[];
  dissentVotes: PlanVote[];
  error: string | null;
  fastMode: boolean;
  crossExaminationMessages: { from: string; to: string; message: string }[];
  startedAt: string;
  completedAt: string | null;
}

export function createSessionState(sessionId: string): SessionState {
  return {
    sessionId,
    phase: "standby",
    taskBrief: null,
    plans: new Map(),
    planVotes: [],
    executorVotes: [],
    winningPlan: null,
    selectedExecutor: null,
    executionOutputs: [],
    bugFixCycles: 0,
    peerReviews: [],
    dissentVotes: [],
    error: null,
    fastMode: false,
    crossExaminationMessages: [],
    startedAt: new Date().toISOString(),
    completedAt: null,
  };
}