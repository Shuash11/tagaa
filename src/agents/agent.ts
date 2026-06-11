import crypto from "crypto";
import { LLMProvider } from "../providers/base.js";
import { TaskBrief, Plan, PlanSchema, PlanVote, PlanVoteSchema, ExecutorVote, ExecutorVoteSchema, PeerReview, PeerReviewSchema } from "../voting/schemas.js";
import { AgentConfig } from "../voting/schemas.js";

export class Agent {
  public readonly name: string;
  public readonly provider: string;
  public readonly model: string;
  public readonly specialty: string;
  public readonly color: string;
  public readonly config: AgentConfig;

  private llmProvider: LLMProvider;

  constructor(config: AgentConfig, llmProvider: LLMProvider) {
    this.name = config.name;
    this.provider = config.provider;
    this.model = config.model;
    this.specialty = config.specialty;
    this.color = config.color;
    this.config = config;
    this.llmProvider = llmProvider;
  }

  private buildSystemPrompt(phase: string, context?: string): string {
    return `You are ${this.name}, an AI agent specialized in ${this.specialty}.
Your provider is ${this.provider} using model ${this.model}.
You are participating in a multi-agent group chat system called TAGAA.

Current phase: ${phase}
${context ? `Context: ${context}` : ""}

Respond only with valid JSON matching the expected schema for this phase. Do not include any text outside the JSON.`;
  }

  async generatePlan(taskBrief: TaskBrief): Promise<Plan> {
    const prompt = `You are generating a plan for the following task:

Task: ${taskBrief.raw_input}
Type: ${taskBrief.classified_type}
Attached files: ${taskBrief.attached_files.join(", ")}

${
  Object.entries(taskBrief.file_contents)
    .map(([path, content]) => `=== ${path} ===\n${content}`)
    .join("\n\n")
}

Generate a structured plan with the following JSON schema:
{
  "agent": "${this.name}",
  "plan_id": "<uuid>",
  "summary": "One sentence describing your approach",
  "steps": [{ "step": 1, "action": "description", "target_file": "optional" }],
  "estimated_complexity": "low|medium|high",
  "risks": ["list of potential failure points"],
  "self_confidence": <0.0-1.0>
}`;

    const systemPrompt = this.buildSystemPrompt("plan_generation", `Specialty: ${this.specialty}\nTask type: ${taskBrief.classified_type}`);
    const response = await this.llmProvider.generate(prompt, systemPrompt);

    const parsed = JSON.parse(response.content);
    parsed.agent = this.name;
    parsed.plan_id = parsed.plan_id || crypto.randomUUID();

    return PlanSchema.parse(parsed);
  }

  async voteOnPlans(plans: Plan[], taskBrief: TaskBrief): Promise<PlanVote> {
    const plansSummary = plans
      .filter((p) => p.agent !== this.name)
      .map(
        (p) =>
          `Plan by ${p.agent} (ID: ${p.plan_id}): ${p.summary}\nSteps: ${p.steps.map((s) => `${s.step}. ${s.action}`).join(" | ")}\nComplexity: ${p.estimated_complexity}\nRisks: ${p.risks.join(", ")}`
      )
      .join("\n\n");

    const prompt = `Task: ${taskBrief.raw_input}

Review the following plans and vote for the best one:

${plansSummary}

Vote with this JSON:
{
  "voter": "${this.name}",
  "voted_for_plan_id": "<uuid of the plan you vote for>",
  "scores": { "correctness": <1-10>, "safety": <1-10>, "simplicity": <1-10>, "completeness": <1-10> },
  "reason": "<why this plan is best>"
}

Scoring dimensions:
- correctness: Does the plan actually solve the problem? (1-10)
- safety: Does it avoid introducing new bugs or risks? (1-10)
- simplicity: Is the solution as simple as possible? (1-10)
- completeness: Does it cover edge cases and error handling? (1-10)`;

    const systemPrompt = this.buildSystemPrompt("plan_vote", `Specialty: ${this.specialty}`);
    const response = await this.llmProvider.generate(prompt, systemPrompt);

    const parsed = JSON.parse(response.content);
    parsed.voter = this.name;

    return PlanVoteSchema.parse(parsed);
  }

  async voteOnExecutor(plans: Plan[], winningPlanId: string, taskBrief: TaskBrief, agents: { name: string; specialty: string }[]): Promise<ExecutorVote> {
    const winningPlan = plans.find((p) => p.plan_id === winningPlanId)!;
    const agentList = agents
      .filter((a) => a.name !== this.name)
      .map((a) => `${a.name} - Specialty: ${a.specialty}`)
      .join("\n");

    const prompt = `Task: ${taskBrief.raw_input}

Winning plan (by ${winningPlan.agent}):
${winningPlan.summary}
Steps: ${winningPlan.steps.map((s) => `${s.step}. ${s.action}`).join(" | ")}
Complexity: ${winningPlan.estimated_complexity}

Available agents and their specialties:
${agentList}

Vote for the best agent to EXECUTE (implement) this plan with this JSON:
{
  "voter": "${this.name}",
  "nominated_executor": "<agent name>",
  "reasoning": "<why this agent is the best executor>",
  "specialty_match_score": <1-10>,
  "context_fit_score": <1-10>
}`;

    const systemPrompt = this.buildSystemPrompt("executor_vote", `Specialty: ${this.specialty}`);
    const response = await this.llmProvider.generate(prompt, systemPrompt);

    const parsed = JSON.parse(response.content);
    parsed.voter = this.name;

    return ExecutorVoteSchema.parse(parsed);
  }

  async proposeBugFix(errorOutput: string, taskBrief: TaskBrief, originalPlan: Plan): Promise<Plan> {
    const prompt = `The following error occurred during execution:

Error:
${errorOutput}

Original task: ${taskBrief.raw_input}
Original plan: ${originalPlan.summary}
Steps: ${originalPlan.steps.map((s) => `${s.step}. ${s.action}`).join(" | ")}

Propose a bug fix plan with this JSON:
{
  "agent": "${this.name}",
  "plan_id": "<uuid>",
  "summary": "One sentence describing the fix",
  "steps": [{ "step": 1, "action": "description", "target_file": "optional" }],
  "estimated_complexity": "low|medium|high",
  "risks": ["list of potential failure points"],
  "self_confidence": <0.0-1.0>
}`;

    const systemPrompt = this.buildSystemPrompt("bug_fix", `Specialty: ${this.specialty}`);
    const response = await this.llmProvider.generate(prompt, systemPrompt);

    const parsed = JSON.parse(response.content);
    parsed.agent = this.name;
    parsed.plan_id = parsed.plan_id || crypto.randomUUID();

    return PlanSchema.parse(parsed);
  }

  async acknowledgeDissent(dissentVotes: PlanVote[], winningPlan: Plan): Promise<string> {
    const dissentSummary = dissentVotes
      .map((v) => `${v.voter} voted for a different plan. Reason: ${v.reason}`)
      .join("\n");

    const prompt = `You are the executor for the winning plan:

Your plan: ${winningPlan.summary}

Dissenting opinions from other agents:
${dissentSummary}

Write a brief acknowledgment of these dissenting views. You will still implement the voted plan, but show that you have considered the alternatives.`;

    const response = await this.llmProvider.generate(prompt);
    return response.content;
  }

  async peerReview(changedFiles: Map<string, string>, taskBrief: TaskBrief): Promise<PeerReview> {
    const files = [...changedFiles.entries()]
      .map(([path, content]) => `=== ${path} ===\n${content}`)
      .join("\n\n");

    const prompt = `Review the following changed files for task: ${taskBrief.raw_input}

${files}

Provide a peer review with this JSON:
{
  "reviewer": "${this.name}",
  "verdict": "approved|changes_requested",
  "comments": [{ "file": "<path>", "line": <number>, "note": "<comment>" }]
}`;

    const response = await this.llmProvider.generate(prompt);
    const parsed = JSON.parse(response.content);
    parsed.reviewer = this.name;

    return PeerReviewSchema.parse(parsed);
  }

  async askClarifyingQuestion(plans: Plan[], targetAgent: string): Promise<string> {
    const targetPlan = plans.find((p) => p.agent === targetAgent);
    if (!targetPlan) return "";

    const prompt = `Review this plan by ${targetAgent}:

${targetPlan.summary}
Steps: ${targetPlan.steps.map((s) => `${s.step}. ${s.action}`).join(" | ")}
Risks: ${targetPlan.risks.join(", ")}

Ask ONE clarifying question about this plan to surface hidden assumptions.`;

    const response = await this.llmProvider.generate(prompt);
    return response.content;
  }

  async respondToClarifyingQuestion(question: string, plan: Plan): Promise<string> {
    const prompt = `You proposed this plan:

${plan.summary}
Steps: ${plan.steps.map((s) => `${s.step}. ${s.action}`).join(" | ")}

Another agent asks: "${question}"

Respond to this question concisely.`;

    const response = await this.llmProvider.generate(prompt);
    return response.content;
  }
}