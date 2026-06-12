package main

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func initialModel() model {
	apiKeys, agents := loadConfig()
	sid, stime, sessionMsgs := loadLatestSession()

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
		sessionID:     sid,
		sessionTime:   stime,
		agents:        agents,
		messages:      msgs,
		apiKeys:       apiKeys,
		models:        make(map[string][]string),
		modelsLoading: make(map[string]bool),
		modelErrors:   make(map[string]string),
		msgAgentIdx:   -1,
		sidebar:       true,
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
		m.isRunning = false
		m.phase = ""
		m.scrollOffset = 0
		clr := lipgloss.Color("#00CED1")
		for i, a := range m.agents {
			if a.Name == msg.agentName {
				clr = agentColors[i%len(agentColors)]
				break
			}
		}
		m.messages = append(m.messages, Message{Kind: MsgAgent, AgentName: msg.agentName, Content: msg.content, Color: clr})
		m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
		return m, nil

	case chatErrMsg:
		m.isRunning = false
		m.phase = ""
		m.scrollOffset = 0
		m.messages = append(m.messages, Message{Kind: MsgError, Content: msg.content})
		m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
		return m, nil

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
		case "ctrl+b":
			m.sidebar = !m.sidebar
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
			m.scrollOffset++
			return m, nil
		case "ctrl+down":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
			return m, nil
		case "ctrl+e":
			if m.sidebar && len(m.agents) > 0 {
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
				m.phase = ""
				m.messages = append(m.messages, Message{Kind: MsgSystem, Content: "Cancelled"})
				m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
			}
			return m, nil
		case "up", "down", "left", "right":
			return m, nil
		case "enter":
			if m.isRunning {
				return m, nil
			}
			if t := strings.TrimSpace(m.input); t != "" {
				m.messages = append(m.messages, Message{Kind: MsgUser, Content: t})
				m.input = ""
				m.isRunning = true
				m.phase = m.nextAgentName()
				m.scrollOffset = 0
				if len(m.agents) > 0 {
					m.msgAgentIdx = (m.msgAgentIdx + 1) % len(m.agents)
				}
				ctx, cancel := context.WithCancel(context.Background())
				m.cancelFn = cancel
				return m, sendChatCmd(m, ctx)
			}
			return m, nil
		case "backspace":
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
				m.input += s
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	sw := 22
	mw := m.w
	if m.sidebar {
		mw = m.w - sw
	}
	if mw < 10 {
		mw = 10
	}

	phaseStr := ""
	if m.phase != "" {
		phaseStr = lipgloss.NewStyle().Foreground(blueC).Render(" [" + m.phase + "]")
		if m.isRunning {
			phaseStr = lipgloss.NewStyle().Foreground(greenC).Render(" [" + m.phase + " ◆]")
		}
	} else {
		phaseStr = lipgloss.NewStyle().Faint(true).Foreground(muteC).Render(" [idle]")
	}
	if m.scrollOffset > 0 {
		phaseStr += lipgloss.NewStyle().Foreground(muteC).Render(fmt.Sprintf(" ↑%d", m.scrollOffset))
	}

	sessionStr := ""
	if m.sessionID > 0 {
		sessionStr = lipgloss.NewStyle().Faint(true).Foreground(muteC).Render(fmt.Sprintf("  #%d", m.sessionID))
	}

	hdr := lipgloss.NewStyle().
		Background(bg).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderC).
		Width(mw - 2).
		PaddingLeft(1).
		Render(lipgloss.NewStyle().Background(bg).Render(
			lipgloss.NewStyle().Bold(true).Foreground(accentC).Render(" TAGAA") +
				sessionStr +
				phaseStr +
				lipgloss.NewStyle().Faint(true).Render("  v0.1.0"),
		))

	inpWidth := mw - 4
	cursor := lipgloss.NewStyle().Background(lipgloss.Color("#E6E6E6")).Render(" ")
	var inpContent string
	if m.isRunning {
		inpContent = lipgloss.NewStyle().Bold(true).Foreground(blueC).Render("◆ Waiting for response…")
	} else {
		inpContent = lipgloss.NewStyle().Bold(true).Foreground(accentC).Render("λ ") +
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E6E6E6")).Render(m.input) +
			cursor
	}
	inp := lipgloss.NewStyle().
		Background(bg).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(blueC).
		Width(inpWidth).
		PaddingLeft(1).
		Render(inpContent)

	msgH := m.h - 6
	if msgH < 1 {
		msgH = 1
	}
	msgW := mw - 2

	if m.settings {
		var msgs string
		if m.settingsTab == 0 {
			msgs = m.keysView(msgW, msgH)
		} else {
			msgs = m.agentsView(msgW, msgH)
		}
		mainCol := lipgloss.JoinVertical(lipgloss.Top, hdr, msgs, inp)
		if m.sidebar {
			return lipgloss.JoinHorizontal(lipgloss.Top, mainCol, m.sideView())
		}
		return mainCol
	}

	total := len(m.messages) - m.scrollOffset
	start := 0
	if total > msgH {
		start = total - msgH
	}
	if start < 0 {
		start = 0
	}

	var body strings.Builder
	for i := start; i < len(m.messages)-m.scrollOffset; i++ {
		rendered := renderMessage(m.messages[i], msgW)
		body.WriteString(rendered)
		body.WriteString("\n")
	}

	lines := strings.Count(body.String(), "\n")
	extra := msgH - lines
	for range extra {
		body.WriteString(strings.Repeat(" ", msgW) + "\n")
	}

	msgs := lipgloss.NewStyle().Background(bg).Width(msgW).Render(body.String())

	if m.sidebarConfig && m.sidebarStep > 0 {
		msgs = m.sidebarDropdown(msgW, msgH)
	}

	mainCol := lipgloss.JoinVertical(lipgloss.Top, hdr, msgs, inp)

	if m.sidebar {
		return lipgloss.JoinHorizontal(lipgloss.Top, mainCol, m.sideView())
	}
	return mainCol
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
