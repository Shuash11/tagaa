package main

import "github.com/charmbracelet/lipgloss"

type provider struct{ id, label string }

type agentCfg struct {
	name     string
	provider string
	model    string
	enabled  bool
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
	w, h          int
	ready         bool
}

type fetchModelsMsg struct {
	provider string
	models   []string
	err      error
}

type chatRespMsg struct {
	agentName string
	content   string
	provider  string
}

type chatErrMsg struct {
	content string
}

type savedConfig struct {
	APIKeys map[string]string `json:"api_keys"`
	Agents  []agentCfg        `json:"agents"`
}
