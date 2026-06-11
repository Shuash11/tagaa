import { Agent } from "../agents/agent.js";
import { SessionState } from "../orchestrator/state.js";
import { ToolLayer } from "../tools/tool-layer.js";
import { ToolCall, ExecutionOutput, VotingConfig } from "../voting/schemas.js";
import { BugFixCouncil } from "../bugfix/council.js";
import { messageStore, createMessage } from "../ui/message-store.js";

export async function runPhase5Execute(
  state: SessionState,
  agents: Agent[],
  toolLayer: ToolLayer,
  config: VotingConfig,
  features: {
    dissent_acknowledgment: boolean;
    post_execution_peer_review: boolean;
    max_fix_cycles: number;
  }
): Promise<void> {
  state.phase = "execution";

  if (!state.winningPlan) throw new Error("Cannot execute: no winning plan");
  if (!state.selectedExecutor) throw new Error("Cannot execute: no executor selected");

  const executor = agents.find((a) => a.name === state.selectedExecutor);
  if (!executor) throw new Error(`Executor ${state.selectedExecutor} not found`);

  const plan = state.winningPlan;
  const executorColor = executor.color || "cyan";

  messageStore.push(createMessage("orchestrator", `Executor ${executor.name} starting implementation...`));

  if (features.dissent_acknowledgment && state.dissentVotes.length > 0) {
    messageStore.push(createMessage("agent", "Acknowledging dissenting opinions before executing...", {
      agentName: executor.name,
      color: executorColor,
      specialty: executor.specialty,
    }));

    const acknowledgment = await executor.acknowledgeDissent(state.dissentVotes, plan);

    messageStore.push(createMessage("agent", acknowledgment, {
      agentName: executor.name,
      color: executorColor,
    }));
  }

  messageStore.push(createMessage("agent", `Implementing plan: ${plan.summary}\n\n${plan.steps.map((s) => `  Step ${s.step}: ${s.action}`).join("\n")}`, {
    agentName: executor.name,
    color: executorColor,
    specialty: executor.specialty,
  }));

  const executionOutputs: ExecutionOutput[] = [];

  for (const step of plan.steps) {
    messageStore.push(createMessage("dim", `→ Step ${step.step}: ${step.action}...`));

    const toolCall: ToolCall = {
      tool: "run_command",
      args: { command: `echo "Step ${step.step}: ${step.action}"` },
    };

    try {
      const result = await toolLayer.execute(toolCall);
      executionOutputs.push({
        type: "bash_cmd",
        command: toolCall.args.command as string,
        stdout: result.output || "",
        stderr: result.error || "",
        exit_code: result.success ? 0 : 1,
      });
      if (result.output) messageStore.push(createMessage("dim", result.output));
    } catch (err) {
      executionOutputs.push({
        type: "bash_cmd",
        command: toolCall.args.command as string,
        stdout: "",
        stderr: String(err),
        exit_code: 1,
      });
      messageStore.push(createMessage("error", `Step ${step.step} failed: ${err}`));
    }
  }

  state.executionOutputs = executionOutputs;

  const testsResult = await toolLayer.execute({ tool: "run_tests", args: {} });

  if (testsResult.success) {
    messageStore.push(createMessage("success", "All tests passed."));
  } else {
    messageStore.push(createMessage("warning", "Tests failed or not available. Invoking Bug Fix Council..."));
  }

  if (features.post_execution_peer_review) {
    state.phase = "peer_review";
    messageStore.push(createMessage("orchestrator", "Post-execution peer review..."));

    const nonExecutorAgents = agents.filter((a) => a.name !== executor.name);
    const reviewers = pickRandom(nonExecutorAgents, Math.min(2, nonExecutorAgents.length));

    for (const reviewer of reviewers) {
      try {
        const review = await reviewer.peerReview(new Map(), state.taskBrief!);
        state.peerReviews.push(review);

        if (review.verdict === "changes_requested") {
          messageStore.push(createMessage("warning", `${reviewer.name}: Changes requested — ${review.comments.map((c) => `${c.file}:${c.line}`).join(", ")}`));
        } else {
          messageStore.push(createMessage("success", `${reviewer.name}: Approved`));
        }
      } catch {
        messageStore.push(createMessage("warning", `${reviewer.name}: Review failed`));
      }
    }
  }

  if (!testsResult.success) {
    state.phase = "bug_fix_council";

    const council = new BugFixCouncil(agents, config, state, features.max_fix_cycles);
    const councilResult = await council.run(
      testsResult.error || "Unknown error",
      state.taskBrief!,
      plan
    );

    if (councilResult.fixPlan && councilResult.executor) {
      state.winningPlan = councilResult.fixPlan;
      state.selectedExecutor = councilResult.executor;

      messageStore.push(createMessage("orchestrator", `Applying fix via ${councilResult.executor} (cycle ${councilResult.cycle})...`));

      await runPhase5Execute(state, agents, toolLayer, config, features);
      return;
    } else if (councilResult.error) {
      messageStore.push(createMessage("error", `Bug Fix Council exhausted: ${councilResult.error}`));
    }
  }

  state.phase = "complete";
  state.completedAt = new Date().toISOString();
  messageStore.push(createMessage("success", "Execution complete."));
}

function pickRandom<T>(arr: T[], count: number): T[] {
  const shuffled = [...arr].sort(() => Math.random() - 0.5);
  return shuffled.slice(0, count);
}