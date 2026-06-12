package main

import (
	"context"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type provider struct{ id, label string }

type agentCfg struct {
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	Enabled        bool   `json:"enabled"`
	IsOrchestrator bool   `json:"is_orchestrator"`
}

type MsgKind int

const (
	MsgUser   MsgKind = iota
	MsgAgent
	MsgSystem
	MsgError
	MsgSuccess
	MsgVote
	MsgPlan
	MsgReview
	MsgDissent
	MsgPhaseDivider
)

type PhaseStatus string

const (
	PhaseIdle     PhaseStatus = "idle"
	PhasePending  PhaseStatus = "pending"
	PhaseActive   PhaseStatus = "active"
	PhaseComplete PhaseStatus = "complete"
	PhaseFailed   PhaseStatus = "failed"
)

type Message struct {
	Kind      MsgKind
	AgentName string
	Color     lipgloss.Color
	Content   string
}

type PlanSummary struct {
	Summary    string
	Steps      []string
	Complexity string
	Confidence float64
	Risks      []string
}

type ReviewLine struct {
	Type    string `json:"type"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type ReviewResult struct {
	Lines   []ReviewLine `json:"lines"`
	Verdict string       `json:"verdict"`
}

type VoteEntry struct {
	Voter    string
	VotedFor string
	Reason   string
	Score    float64
}

type VoteResult struct {
	Phase   string
	Entries []VoteEntry
	Winner  string
	Scores  map[string]float64
}

// PendingAgent tracks an agent that is currently thinking
type PendingAgent struct {
	Name      string
	Color     lipgloss.Color
	Elapsed   time.Duration
	StartedAt time.Time
}

// AgentStatusType holds full status information for an agent
type AgentStatusType struct {
	Name      string
	Color     lipgloss.Color
	Status    string   // "thinking", "done", "error"
	Elapsed   time.Duration
	StartedAt time.Time
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type thinkingTickMsg struct{}

// AgentResponse holds a single agent's completed response
type AgentResponse struct {
	AgentName string
	Content   string
	Color     lipgloss.Color
}

type PipelineState struct {
	Intake      PhaseStatus
	Planning    PhaseStatus
	PlanVote    PhaseStatus
	ExecVote    PhaseStatus
	Execution   PhaseStatus
	Review      PhaseStatus
}

type PhaseName string

const (
	PhaseIntake    PhaseName = "intake"
	PhasePlanning  PhaseName = "planning"
	PhasePlanVote  PhaseName = "plan_vote"
	PhaseExecVote  PhaseName = "exec_vote"
	PhaseExecution PhaseName = "execution"
	PhaseReview    PhaseName = "review"
)

type PlanResponse struct {
	AgentName string
	Color     lipgloss.Color
	Plan      string
	Steps     []string
}

// PipelineActionMsg is sent from pipeline to the TUI
type PipelineActionMsg struct {
	Action  string
	Content interface{}
}

// internal message for pipeline phase completion
type pipelineBatchMsg struct {
	phase    string
	messages []Message
	state    PipelineState
	done     bool
}

type pipelineTickMsg struct{}

func (ps PipelineState) String() string {
	type phaseDef struct {
		s   PhaseStatus
		lbl string
	}
	phases := []phaseDef{
		{ps.Intake, "Intake"},
		{ps.Planning, "Planning"},
		{ps.PlanVote, "Vote"},
		{ps.ExecVote, "Exec"},
		{ps.Execution, "Execute"},
		{ps.Review, "Review"},
	}
	var result string
	for i, p := range phases {
		if i > 0 {
			result += " → "
		}
		var dot string
		var clr lipgloss.Color
		switch p.s {
		case PhaseActive:
			dot = "●"
			clr = lipgloss.Color("#00FF00")
		case PhaseComplete:
			dot = "✓"
			clr = lipgloss.Color("#00FFFF")
		case PhaseFailed:
			dot = "✗"
			clr = lipgloss.Color("#FF0000")
		default:
			dot = "○"
			clr = muteC
		}
		result += lipgloss.NewStyle().Foreground(clr).Render(dot + " " + p.lbl)
	}
	return result
}

func (ps PipelineState) ActivePhase() string {
	phases := []struct {
		s   PhaseStatus
		lbl string
	}{
		{ps.Intake, "Intake"},
		{ps.Planning, "Planning"},
		{ps.PlanVote, "Voting"},
		{ps.ExecVote, "Selecting Executor"},
		{ps.Execution, "Executing"},
		{ps.Review, "Reviewing"},
	}
	for _, p := range phases {
		if p.s == PhaseActive {
			return p.lbl
		}
	}
	return ""
}

type model struct {
	agents        []agentCfg
	messages      []Message
	input         string
	apiKeys       map[string]string
	models        map[string][]string
	modelsLoading map[string]bool
	modelErrors   map[string]string
	phase             string
	pipeline          PipelineState
	cancelledPhase    string
	isRunning         bool
	cancelledAgentCount int
	scrollOffset      int
	sessionID     int
	sessionTime   string
	pendingAgents  []PendingAgent
	agentResponses []AgentResponse
	sidebar        bool
	sidebarConfig bool
	sidebarSel    int
	sidebarStep   int
	sidebarCur    int
	sidebarProv   string
	settings      bool
	settingsTab   int
	setCur        int
	setEdit       bool
	setKey        string
	agentCur      int
	agentEdit     bool
	agentField    int
	agentTemp     string
	cancelFn      context.CancelFunc
	streamText    string
	streamPos     int
	cmdMode       bool
	cmdCur        int
	cmdSessions   []Session
	thinking      bool
	w, h          int
	ready         bool
	spinnerIdx      int
	planExpanded    map[int]bool
	totalTokens     int
	showTokenEstimate bool
	pipelineCh      chan pipelineBatchMsg
}

type fetchModelsMsg struct {
	provider string
	models   []string
	err      error
}

type chatRespMsg struct {
	agentName  string
	content    string
	provider   string
	toolResult string
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatErrMsg struct {
	agentName string
	content   string
}

type streamTickMsg struct{}

type savedConfig struct {
	APIKeys           map[string]string `json:"api_keys"`
	Agents            []agentCfg        `json:"agents"`
	ShowTokenEstimate bool              `json:"show_token_estimate"`
}
