package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Pipeline struct {
	phase           PhaseName
	agents          []agentCfg
	apiKeys         map[string]string
	history         []Message
	plans           []PlanResponse
	state           PipelineState
	vote            *VoteResult
	executionOutput string
}

func NewPipeline(agents []agentCfg, apiKeys map[string]string) *Pipeline {
	return &Pipeline{
		agents:  agents,
		apiKeys: apiKeys,
	}
}

func (p *Pipeline) Run(ctx context.Context, task string, ch chan<- pipelineBatchMsg) {
	defer close(ch)

	p.state.Intake = PhaseActive
	ch <- p.batch("intake", p.runIntakePhase(task))
	p.state.Intake = PhaseComplete
	if ctx.Err() != nil {
		return
	}

	if !isTaskRequest(task) {
		p.state.Planning = PhaseComplete
		p.state.PlanVote = PhaseComplete
		p.state.ExecVote = PhaseComplete
		p.state.Execution = PhaseComplete
		p.state.Review = PhaseComplete
		ch <- p.batch("chat", p.runChatResponse(ctx, task))
		return
	}

	p.state.Planning = PhaseActive
	ch <- p.batch("planning", p.runPlanningPhase(ctx, task))
	p.state.Planning = PhaseComplete
	if ctx.Err() != nil {
		return
	}

	if len(p.agents) == 1 {
		p.state.PlanVote = PhaseComplete
		p.state.ExecVote = PhaseComplete

		p.state.Execution = PhaseActive
		ch <- p.batch("execution", p.runExecutionPhase(ctx))
		p.state.Execution = PhaseComplete
		if ctx.Err() != nil {
			return
		}

		if p.executionOutput != "" {
			p.state.Review = PhaseActive
			ch <- p.batch("review", p.runReviewPhase(ctx))
			p.state.Review = PhaseComplete
		} else {
			p.state.Review = PhaseComplete
		}
		return
	}

	p.state.PlanVote = PhaseActive
	ch <- p.batch("plan_vote", p.runPlanVotePhase(ctx))
	p.state.PlanVote = PhaseComplete
	if ctx.Err() != nil {
		return
	}

	p.state.ExecVote = PhaseActive
	ch <- p.batch("exec_vote", p.runExecVotePhase(ctx))
	p.state.ExecVote = PhaseComplete
	if ctx.Err() != nil {
		return
	}

	p.state.Execution = PhaseActive
	ch <- p.batch("execution", p.runExecutionPhase(ctx))
	p.state.Execution = PhaseComplete
	if ctx.Err() != nil {
		return
	}

	p.state.Review = PhaseActive
	ch <- p.batch("review", p.runReviewPhase(ctx))
	p.state.Review = PhaseComplete
}

func (p *Pipeline) batch(phase string, msgs []Message) pipelineBatchMsg {
	return pipelineBatchMsg{phase: phase, messages: msgs, state: p.state}
}

func (p *Pipeline) runIntakePhase(task string) []Message {
	var msgs []Message
	msgs = append(msgs, Message{Kind: MsgPhaseDivider, Content: " INTAKE "})
	msgs = append(msgs, Message{Kind: MsgSystem, Content: "Task: " + task})
	files := extractFilePaths(task)
	if len(files) > 0 {
		msgs = append(msgs, Message{Kind: MsgSystem, Content: "Referenced files:"})
		for _, f := range files {
			msgs = append(msgs, Message{Kind: MsgSystem, Content: "  " + f})
		}
	}
	if !isTaskRequest(task) {
		msgs = append(msgs, Message{Kind: MsgSystem, Content: "Detected: chat message → direct response"})
	}
	msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
	return msgs
}

func isTaskRequest(task string) bool {
	lower := strings.ToLower(task)
	if strings.HasPrefix(lower, "!") {
		return true
	}
	filePatterns := []string{".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".java", ".c", ".cpp", ".h", ".css", ".yaml", ".yml", ".json", ".md", ".toml", "```"}
	for _, p := range filePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	imperative := []string{
		"implement ", "refactor ", "fix the bug", "write a ", "create a ",
		"build a ", "deploy ", "debug ", "optimize ", "configure ",
	}
	for _, w := range imperative {
		if strings.HasPrefix(lower, w) {
			return true
		}
	}
	return false
}

func (p *Pipeline) runChatResponse(ctx context.Context, task string) []Message {
	var msgs []Message
	msgs = append(msgs, Message{Kind: MsgPhaseDivider, Content: " CHAT "})

	if len(p.agents) == 0 {
		msgs = append(msgs, Message{Kind: MsgError, Content: "No agents available"})
		return msgs
	}

	agent := p.agents[0]
	system := "You are a helpful AI assistant. Be conversational and concise."
	userMsg := task + p.conversationContext(5)
	text, err := queryAgentSimple(ctx, agent, p.apiKeys, system, userMsg)
	if err != nil {
		msgs = append(msgs, Message{Kind: MsgError, Content: agent.Name + ": " + err.Error()})
		msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
		return msgs
	}

	msgs = append(msgs, Message{Kind: MsgAgent, AgentName: agent.Name, Content: text, Color: p.agentColor(agent.Name)})
	msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
	return msgs
}

func (p *Pipeline) runPlanningPhase(ctx context.Context, task string) []Message {
	type result struct {
		name string
		clr  lipgloss.Color
		text string
		err  error
	}

	ch := make(chan result, len(p.agents))
	for i, agent := range p.agents {
		go func(a agentCfg, idx int) {
			system := "You are a technical planning agent. Create a detailed plan for the task. Output a clear summary and numbered steps."
			text, err := queryAgentSimple(ctx, a, p.apiKeys, system, "Plan this task: "+task+p.conversationContext(5))
			ch <- result{name: a.Name, clr: p.agentColor(a.Name), text: text, err: err}
		}(agent, i)
	}

	var msgs []Message
	msgs = append(msgs, Message{Kind: MsgPhaseDivider, Content: " PLANNING "})
	for range p.agents {
		r := <-ch
		if r.err != nil {
			msgs = append(msgs, Message{Kind: MsgError, Content: r.name + ": " + r.err.Error()})
			continue
		}
		p.plans = append(p.plans, PlanResponse{
			AgentName: r.name, Color: r.clr, Plan: r.text,
		})
		msgs = append(msgs, Message{Kind: MsgPlan, AgentName: r.name, Content: r.text, Color: r.clr})
	}
	msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
	return msgs
}

func (p *Pipeline) runPlanVotePhase(ctx context.Context) []Message {
	prompt := "Review the following plans and vote for the best one. " +
		"Consider completeness, feasibility, and correctness.\n" +
		"Respond with: VOTE: <agent name>\n" +
		"Then explain your reasoning briefly." + p.plansSummary()

	type voteResult struct {
		voter   string
		vote    string
		rawText string
		err     error
	}

	ch := make(chan voteResult, len(p.agents))
	for _, agent := range p.agents {
		go func(a agentCfg) {
			text, err := queryAgentSimple(ctx, a, p.apiKeys, "You are evaluating plans.", prompt)
			if err != nil {
				ch <- voteResult{voter: a.Name, err: err}
				return
			}
			voted := p.parseVote(text, p.plans)
			ch <- voteResult{voter: a.Name, vote: voted, rawText: text}
		}(agent)
	}

	scores := make(map[string]float64)
	var entries []VoteEntry
	for range p.agents {
		r := <-ch
		if r.err != nil {
			continue
		}
		if r.vote != "" {
			scores[r.vote]++
			reason := ""
			if r.rawText != "" {
				reason = extractReasonExcerpt(r.rawText)
			}
			entries = append(entries, VoteEntry{Voter: r.voter, VotedFor: r.vote, Reason: reason})
		}
	}

	winner, maxScore := "", 0.0
	for name, sc := range scores {
		if sc > maxScore {
			maxScore = sc
			winner = name
		}
	}

	result := VoteResult{Phase: "plan", Entries: entries, Winner: winner, Scores: scores}
	p.vote = &result
	data, _ := json.Marshal(result)

	var msgs []Message
	msgs = append(msgs, Message{Kind: MsgPhaseDivider, Content: " PLAN VOTE "})
	msgs = append(msgs, Message{Kind: MsgVote, Content: string(data)})
	msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
	return msgs
}

func (p *Pipeline) runExecVotePhase(ctx context.Context) []Message {
	prompt := "Based on the plans created, who should execute the task?\n" +
		"Consider each agent's expertise and the plan quality.\n" +
		"Respond with: VOTE: <agent name>" + p.plansSummary()

	type voteResult struct {
		voter   string
		vote    string
		rawText string
		err     error
	}

	ch := make(chan voteResult, len(p.agents))
	for _, agent := range p.agents {
		go func(a agentCfg) {
			text, err := queryAgentSimple(ctx, a, p.apiKeys, "You are selecting an executor.", prompt)
			if err != nil {
				ch <- voteResult{voter: a.Name, err: err}
				return
			}
			voted := p.parseVote(text, p.plans)
			ch <- voteResult{voter: a.Name, vote: voted, rawText: text}
		}(agent)
	}

	scores := make(map[string]float64)
	var entries []VoteEntry
	for range p.agents {
		r := <-ch
		if r.err != nil {
			continue
		}
		if r.vote != "" {
			scores[r.vote]++
			reason := "executor selection"
			if r.rawText != "" {
				if excerpt := extractReasonExcerpt(r.rawText); excerpt != "" {
					reason = excerpt
				}
			}
			entries = append(entries, VoteEntry{Voter: r.voter, VotedFor: r.vote, Reason: reason})
		}
	}

	winner, maxScore := "", 0.0
	for name, sc := range scores {
		if sc > maxScore {
			maxScore = sc
			winner = name
		}
	}

	result := VoteResult{Phase: "executor", Entries: entries, Winner: winner, Scores: scores}
	p.vote = &result
	data, _ := json.Marshal(result)

	var msgs []Message
	msgs = append(msgs, Message{Kind: MsgPhaseDivider, Content: " EXECUTOR VOTE "})
	msgs = append(msgs, Message{Kind: MsgVote, Content: string(data)})
	msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
	return msgs
}

func (p *Pipeline) runExecutionPhase(ctx context.Context) []Message {
	var executor string
	if p.vote != nil && p.vote.Winner != "" {
		executor = p.vote.Winner
	} else if len(p.plans) > 0 {
		executor = p.plans[0].AgentName
	} else {
		executor = p.agents[0].Name
	}

	var agent agentCfg
	for _, a := range p.agents {
		if a.Name == executor {
			agent = a
			break
		}
	}

	var msgs []Message
	msgs = append(msgs, Message{Kind: MsgPhaseDivider, Content: " EXECUTION "})
	msgs = append(msgs, Message{Kind: MsgSystem, Content: "Selected executor: " + executor})
	msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})

	system := "You are the executor. Implement the plan. Use tools as needed and provide clear output."
	text, err := queryAgentSimple(ctx, agent, p.apiKeys, system,
		"Execute the following plan:\n"+p.plansSummary()+"\n\nProvide your implementation output.")
	if err != nil {
		msgs = append(msgs, Message{Kind: MsgError, Content: executor + ": " + err.Error()})
		msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
		return msgs
	}

	msgs = append(msgs, Message{Kind: MsgAgent, AgentName: executor, Content: text, Color: p.agentColor(executor)})
	msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
	p.executionOutput = text
	return msgs
}

func (p *Pipeline) runReviewPhase(ctx context.Context) []Message {
	var executor string
	if p.vote != nil && p.vote.Winner != "" {
		executor = p.vote.Winner
	} else if len(p.plans) > 0 {
		executor = p.plans[0].AgentName
	} else {
		executor = p.agents[0].Name
	}

	var reviewAgents []agentCfg
	for _, a := range p.agents {
		if a.Name != executor {
			reviewAgents = append(reviewAgents, a)
		}
	}
	if len(reviewAgents) == 0 {
		reviewAgents = p.agents
	}

	type reviewResult struct {
		name string
		clr  lipgloss.Color
		text string
		err  error
	}

	ch := make(chan reviewResult, len(reviewAgents))
	for _, a := range reviewAgents {
		go func(agent agentCfg) {
			system := "You are a code reviewer. Review the execution output for correctness, completeness, and potential issues."
			text, err := queryAgentSimple(ctx, agent, p.apiKeys, system,
				"Review this execution output and provide feedback:\n\n"+
					"Executor: "+executor+"\n"+
					"Execution output:\n"+p.executionOutput+"\n\n"+
					"Plans overview: "+p.plansSummary()+"\n"+
					"Provide a verdict: pass or needs_fix. List any issues found.")
			ch <- reviewResult{name: agent.Name, clr: p.agentColor(agent.Name), text: text, err: err}
		}(a)
	}

	var msgs []Message
	msgs = append(msgs, Message{Kind: MsgPhaseDivider, Content: " REVIEW "})
	for range reviewAgents {
		r := <-ch
		if r.err != nil {
			msgs = append(msgs, Message{Kind: MsgError, Content: r.name + ": " + r.err.Error()})
			continue
		}
		msgs = append(msgs, Message{Kind: MsgReview, AgentName: r.name, Content: r.text, Color: r.clr})
	}
	msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
	return msgs
}
