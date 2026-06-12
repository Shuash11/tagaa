package main

import (
	"context"

	"github.com/charmbracelet/lipgloss"
)

type provider struct{ id, label string }

type agentCfg struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Enabled  bool   `json:"enabled"`
}

type MsgKind int

const (
	MsgUser   MsgKind = iota
	MsgAgent
	MsgSystem
	MsgError
	MsgSuccess
	MsgVote
)

type Message struct {
	Kind      MsgKind
	AgentName string
	Color     lipgloss.Color
	Content   string
}

type model struct {
	agents        []agentCfg
	messages      []Message
	input         string
	apiKeys       map[string]string
	models        map[string][]string
	modelsLoading map[string]bool
	modelErrors   map[string]string
	phase         string
	isRunning     bool
	scrollOffset  int
	sessionID     int
	sessionTime   string
	msgAgentIdx   int
	sidebar       bool
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
	w, h          int
	ready         bool
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
	content string
}

type streamTickMsg struct{}

type savedConfig struct {
	APIKeys map[string]string `json:"api_keys"`
	Agents  []agentCfg        `json:"agents"`
}
