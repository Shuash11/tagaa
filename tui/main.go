package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func initialModel() model {
	apiKeys, agents, showTokenEstimate := loadConfig()

	var freshStart bool
	for _, arg := range os.Args[1:] {
		if arg == "--fresh" || arg == "-f" {
			freshStart = true
			break
		}
	}

	sid, stime, sessionMsgs := 0, "", []Message(nil)
	if !freshStart {
		sid, stime, sessionMsgs = loadLatestSession()
	}

	var msgs []Message
	if len(sessionMsgs) > 0 {
		msgs = sessionMsgs
		msgs = append(msgs, Message{Kind: MsgSystem, Content: ""})
	} else {
		msgs = []Message{
			{Kind: MsgSystem, Content: "◆ TAGAA v0.1.0 — Terminal Autonomous Group AI Assistant"},
			{Kind: MsgSystem, Content: "◆ API keys and agents loaded from " + configFile},
			{Kind: MsgSystem, Content: "  Ctrl+B toggle sidebar · Ctrl+S settings · Ctrl+E config"},
			{Kind: MsgSystem, Content: ""},
		}
	}

	return model{
		sessionID:         sid,
		sessionTime:       stime,
		agents:            agents,
		messages:          msgs,
		apiKeys:           apiKeys,
		models:            make(map[string][]string),
		modelsLoading:     make(map[string]bool),
		modelErrors:       make(map[string]string),
		sidebar:           true,
		showTokenEstimate: showTokenEstimate,
		planExpanded:      make(map[int]bool),
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for id, key := range m.apiKeys {
		if key != "" {
			cmds = append(cmds, fetchModelsCmd(id, key))
		}
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.ready = true
		return m, nil

	case fetchModelsMsg:
		m.modelsLoading[msg.provider] = false
		if msg.err != nil {
			m.modelErrors[msg.provider] = msg.err.Error()
			m.models[msg.provider] = nil
		} else {
			delete(m.modelErrors, msg.provider)
			m.models[msg.provider] = msg.models
		}
		return m, nil

	case chatRespMsg:
		if !m.isRunning {
			return m, nil
		}
		// Remove agent from pendingAgents
		for i, pa := range m.pendingAgents {
			if pa.Name == msg.agentName {
				m.pendingAgents = append(m.pendingAgents[:i], m.pendingAgents[i+1:]...)
				break
			}
		}
		clr := lipgloss.Color("#00CED1")
		for i, a := range m.agents {
			if a.Name == msg.agentName {
				clr = agentColors[i%len(agentColors)]
				break
			}
		}
		if msg.toolResult != "" {
			m.messages = append(m.messages, Message{Kind: MsgSystem, Content: msg.toolResult})
			m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
		}
		if msg.content == "" && msg.toolResult == "" {
			m.messages = append(m.messages, Message{Kind: MsgAgent, AgentName: msg.agentName, Content: "(empty response)", Color: clr})
			m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
			if len(m.pendingAgents) == 0 {
				if len(m.agentResponses) > 0 {
					var cmd tea.Cmd
					m, cmd = startNextStream(m)
					return m, cmd
				}
				m.isRunning = false
				m.phase = ""
			}
			return m, nil
		}
		m.totalTokens += len(msg.content) / 4
		m.agentResponses = append(m.agentResponses, AgentResponse{
			AgentName: msg.agentName,
			Content:   msg.content,
			Color:     clr,
		})
		if len(m.pendingAgents) == 0 && (m.streamText == "" || m.streamPos >= len(m.streamText)) {
			var cmd tea.Cmd
			m, cmd = startNextStream(m)
			return m, cmd
		}
		return m, nil

	case thinkingTickMsg:
		if !m.isRunning || len(m.pendingAgents) == 0 {
			return m, nil
		}
		m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
		now := time.Now()
		for i := range m.pendingAgents {
			m.pendingAgents[i].Elapsed = now.Sub(m.pendingAgents[i].StartedAt)
		}
		return m, thinkingTickCmd()

	case streamTickMsg:
		if !m.isRunning || m.streamText == "" {
			return m, nil
		}
		m.streamPos += 3
		if m.streamPos >= len(m.streamText) {
			m.streamPos = len(m.streamText)
		}
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Kind == MsgAgent {
				m.messages[i].Content = m.streamText[:m.streamPos]
				break
			}
		}
		if m.streamPos >= len(m.streamText) {
			m.streamText = ""
			m.streamPos = 0
			if len(m.agentResponses) > 0 {
				var cmd tea.Cmd
				m, cmd = startNextStream(m)
				return m, cmd
			}
			m.isRunning = false
			m.phase = ""
			return m, nil
		}
		return m, streamTickCmd()

	case chatErrMsg:
		if msg.agentName != "" {
			for i, pa := range m.pendingAgents {
				if pa.Name == msg.agentName {
					m.pendingAgents = append(m.pendingAgents[:i], m.pendingAgents[i+1:]...)
					break
				}
			}
		}
		m.messages = append(m.messages, Message{Kind: MsgError, Content: msg.content})
		m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
		if len(m.pendingAgents) == 0 && (m.streamText == "" || m.streamPos >= len(m.streamText)) {
			if len(m.agentResponses) > 0 {
				var cmd tea.Cmd
				m, cmd = startNextStream(m)
				return m, cmd
			}
			m.isRunning = false
			m.phase = ""
		}
		return m, nil

	case pipelineBatchMsg:
		if msg.done {
			m.isRunning = false
			m.pipelineCh = nil
			m.pendingAgents = nil
			if msg.messages != nil {
				m.messages = append(m.messages, msg.messages...)
			}
			m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
			m.pipeline = msg.state
			m.phase = ""
			return m, nil
		}
		m.messages = append(m.messages, msg.messages...)
		m.pipeline = msg.state
		m.phase = msg.phase
		return m, pipelinePollCmd(m.pipelineCh)

	case pipelineTickMsg:
		if m.pipelineCh == nil {
			return m, nil
		}
		return m, pipelinePollCmd(m.pipelineCh)

	case tea.KeyMsg:
		if m.settings {
			return m.updSettings(msg)
		}
		if m.sidebarConfig {
			return m.updSidebarConfig(msg)
		}
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			saveSessions(m.messages)
			return m, tea.Quit
		case "esc":
			if m.cmdMode {
				m.cmdMode = false
			}
			return m, nil
		case "ctrl+b":
			m.sidebar = !m.sidebar
			return m, nil
		case "ctrl+n":
			saveSessions(m.messages)
			m.sessionID = 0
			m.sessionTime = ""
			m.messages = []Message{
				{Kind: MsgSystem, Content: "◆ New session"},
				{Kind: MsgSystem, Content: "◆ Ctrl+N new session · Ctrl+S settings · Ctrl+B sidebar"},
				{Kind: MsgSystem, Content: ""},
			}
			m.totalTokens = 0
			m.scrollOffset = 0
			m.pipeline = PipelineState{}
			return m, nil
		case "ctrl+t":
			m.thinking = !m.thinking
			return m, nil
		case "ctrl+s":
			m.settings = true
			m.settingsTab = 0
			m.setCur = 0
			m.setEdit = false
			m.setKey = ""
			m.agentCur = 0
			m.agentEdit = false
			return m, nil
		case "ctrl+up":
			m.scrollOffset += 3
			if m.scrollOffset > 99999 {
				m.scrollOffset = 99999
			}
			return m, nil
		case "ctrl+down":
			if m.scrollOffset >= 3 {
				m.scrollOffset -= 3
			} else {
				m.scrollOffset = 0
			}
			return m, nil
		case "pgup":
			m.scrollOffset += m.h / 2
			if m.scrollOffset > 99999 {
				m.scrollOffset = 99999
			}
			return m, nil
		case "pgdown":
			n := m.h / 2
			if m.scrollOffset >= n {
				m.scrollOffset -= n
			} else {
				m.scrollOffset = 0
			}
			return m, nil
		case "ctrl+e":
			if len(m.agents) > 0 {
				m.sidebar = true
				m.sidebarConfig = true
				m.sidebarSel = 0
				m.sidebarStep = 0
			}
			return m, nil
		case "ctrl+x":
			if m.isRunning && m.cancelFn != nil {
				m.cancelFn()
				m.cancelFn = nil
				m.isRunning = false
				m.pipelineCh = nil
				m.phase = ""
				m.streamText = ""
				m.streamPos = 0
				m.pendingAgents = nil
				m.agentResponses = nil
				m.messages = append(m.messages, Message{Kind: MsgSystem, Content: "Cancelled"})
				m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
			}
			return m, nil
		case "up", "down":
			if m.cmdMode {
				n := len(m.cmdSessions)
				if n == 0 {
					return m, nil
				}
				if msg.String() == "up" {
					m.cmdCur--
					if m.cmdCur < 0 {
						m.cmdCur = n - 1
					}
				} else {
					m.cmdCur++
					if m.cmdCur >= n {
						m.cmdCur = 0
					}
				}
				return m, nil
			}
			return m, nil
		case "left", "right":
			return m, nil
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			s := msg.String()
			if m.cmdMode || m.settings || m.sidebarConfig {
				return m, nil
			}
			if m.planExpanded == nil {
				m.planExpanded = make(map[int]bool)
			}
			target := int(s[0] - '0')
			planCount := 0
			for i, msg := range m.messages {
				if msg.Kind == MsgPlan {
					planCount++
					if planCount == target {
						m.planExpanded[i] = !m.planExpanded[i]
						return m, nil
					}
				}
			}
			return m, nil
		case "enter":
			if m.cmdMode {
				return m, nil
			}
			if m.isRunning {
				return m, nil
			}
			if t := strings.TrimSpace(m.input); t != "" {
				m.messages = append(m.messages, Message{Kind: MsgUser, Content: t})
				m.input = ""

				ready := false
				for _, a := range m.agents {
					if a.Enabled && a.Provider != "" && a.Model != "" && m.apiKeys[a.Provider] != "" {
						ready = true
						break
					}
				}
				if !ready {
					m.messages = append(m.messages, Message{Kind: MsgError, Content: "No ready agents configured"})
					m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
					return m, nil
				}

m.pendingAgents = nil
				m.agentResponses = nil
				m.pipelineCh = make(chan pipelineBatchMsg, 10)
				m.isRunning = true
				m.phase = "pipeline"
				m.scrollOffset = 0
				m.streamText = ""
				m.streamPos = 0
				m.totalTokens += len(t) / 4
				ctx, cancel := context.WithCancel(context.Background())
				m.cancelFn = cancel
				return m, sendGroupThinkCmd(m, ctx)
			}
			return m, nil
		case "backspace":
			if m.cmdMode {
				m.cmdMode = false
				return m, nil
			}
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
			return m, nil
		case " ":
			m.input += " "
			return m, nil
		default:
			s := msg.String()
			if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
				if m.cmdMode {
					if s == "d" && m.cmdCur < len(m.cmdSessions) {
						id := m.cmdSessions[m.cmdCur].ID
						deleteSession(id)
						m.cmdSessions = loadAllSessions().Sessions
						if m.cmdCur >= len(m.cmdSessions) && m.cmdCur > 0 {
							m.cmdCur--
						}
					}
					return m, nil
				}
				if m.input == "" && s == "/" {
					m.cmdMode = true
					m.cmdCur = 0
					m.cmdSessions = loadAllSessions().Sessions
					return m, nil
				}
				m.input += s
			}
		}
	}
	return m, nil
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
