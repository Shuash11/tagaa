package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	sw := 23
	mw := m.w
	if m.sidebar {
		mw = m.w - sw
	}
	if mw < 10 {
		mw = 10
	}

	sessionStr := ""
	if m.sessionID > 0 {
		sessionStr = lipgloss.NewStyle().Faint(true).Foreground(muteC).Render(fmt.Sprintf(" #%d", m.sessionID))
	}

	pipelineStr := "  │  " + m.pipeline.String()

	extraStr := ""
	if m.thinking {
		extraStr += lipgloss.NewStyle().Foreground(accentC).Render(" 🧠")
	}
	if m.scrollOffset > 0 {
		extraStr += lipgloss.NewStyle().Foreground(accentC).Render(fmt.Sprintf(" ↑%d", m.scrollOffset))
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
				pipelineStr +
				extraStr +
				lipgloss.NewStyle().Faint(true).Render("  │  v0.1.0"),
		))

	inpWidth := mw - 4
	cursor := lipgloss.NewStyle().Background(lipgloss.Color("#E6E6E6")).Render(" ")
	var inpContent string
	if m.orchMode {
		// orchestrator chat input area
		inpContent = lipgloss.NewStyle().Bold(true).Foreground(accentC).Render("Orch> ") +
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E6E6E6")).Render(m.orchInput) +
			cursor
	} else if m.isRunning {
		phase := m.pipeline.ActivePhase()
		if phase != "" {
			inpContent = lipgloss.NewStyle().Bold(true).Foreground(blueC).Render("◆ " + phase + "…")
		} else if m.phase != "" {
			inpContent = lipgloss.NewStyle().Bold(true).Foreground(blueC).Render("◆ " + m.phase + "…")
		} else {
			inpContent = lipgloss.NewStyle().Bold(true).Foreground(blueC).Render("◆ Waiting for response…")
		}
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

	if m.cmdMode {
		msgs := m.cmdModeView(msgW, msgH)
		mainCol := lipgloss.JoinVertical(lipgloss.Top, hdr, msgs, inp)
		if m.sidebar {
			return lipgloss.JoinHorizontal(lipgloss.Top, mainCol, m.sideView())
		}
		return mainCol
	}

	// Orchestrator chat view
	if m.orchMode {
		// show orchMessages in main area
		var allLines []string
		for _, msg := range m.orchMessages {
			r := m.renderMessage(msg, 0, msgW)
			allLines = append(allLines, strings.Split(r, "\n")...)
		}
		// pad
		for i := 0; i < msgH-len(allLines); i++ {
			allLines = append(allLines, strings.Repeat(" ", msgW))
		}
		var body strings.Builder
		for _, line := range allLines[:msgH] {
			body.WriteString(line + "\n")
		}
		msgs := lipgloss.NewStyle().Background(bg).Width(msgW).Render(body.String())
		mainCol := lipgloss.JoinVertical(lipgloss.Top, hdr, msgs, inp)
		if m.sidebar {
			return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Height(m.h).Width(mw).Render(mainCol), m.sideView())
		}
		return mainCol
	}

	var allLines []string

	frame := ""
	if len(m.pendingAgents) > 0 {
		frame = spinnerFrames[m.spinnerIdx]
	}

	doneNames := make(map[string]bool)
	for _, resp := range m.agentResponses {
		doneNames[resp.AgentName] = true
	}

	for _, pa := range m.pendingAgents {
		if doneNames[pa.Name] {
			elapsed := fmt.Sprintf("(%.1fs)", pa.Elapsed.Seconds())
			allLines = append(allLines, lipgloss.NewStyle().Foreground(pa.Color).Render("  ✓ "+pa.Name+"  Done  "+elapsed))
		} else {
			elapsed := fmt.Sprintf("(%.1fs)", pa.Elapsed.Seconds())
			allLines = append(allLines, lipgloss.NewStyle().Foreground(pa.Color).Render("  🤔 "+pa.Name+"  "+frame+" Generating...  "+elapsed))
		}
	}
	if len(m.pendingAgents) > 0 {
		allLines = append(allLines, "")
	}

	for i, msg := range m.messages {
		r := m.renderMessage(msg, i, msgW)
		allLines = append(allLines, strings.Split(r, "\n")...)
	}

	maxOff := len(allLines) - msgH
	if maxOff < 0 {
		maxOff = 0
	}
	if m.scrollOffset > maxOff {
		m.scrollOffset = maxOff
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	end := len(allLines) - m.scrollOffset
	start := end - msgH
	if start < 0 {
		start = 0
	}
	if end > len(allLines) {
		end = len(allLines)
	}
	if start > end {
		start = end
	}

	var body strings.Builder
	for _, line := range allLines[start:end] {
		body.WriteString(line)
		body.WriteString("\n")
	}
	for i := end - start; i < msgH; i++ {
		body.WriteString(strings.Repeat(" ", msgW) + "\n")
	}

	msgs := lipgloss.NewStyle().Background(bg).Width(msgW).Render(body.String())

	if m.sidebarConfig && m.sidebarStep > 0 {
		msgs = m.sidebarDropdown(msgW, msgH)
	}

	mainCol := lipgloss.JoinVertical(lipgloss.Top, hdr, msgs, inp)

	if m.sidebar {
		return lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Height(m.h).Width(mw).Render(mainCol),
			m.sideView(),
		)
	}
	return mainCol
}

func startNextStream(m model) (model, tea.Cmd) {
	if len(m.agentResponses) == 0 {
		m.isRunning = false
		m.phase = ""
		return m, nil
	}
	resp := m.agentResponses[0]
	m.agentResponses = m.agentResponses[1:]
	m.streamText = resp.Content
	m.streamPos = 0
	m.phase = resp.AgentName
	m.messages = append(m.messages, Message{Kind: MsgAgent, AgentName: resp.AgentName, Content: "", Color: resp.Color})
	m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
	m.scrollOffset = 0
	return m, streamTickCmd()
}

func streamTickCmd() tea.Cmd {
	return tea.Tick(15*time.Millisecond, func(t time.Time) tea.Msg {
		return streamTickMsg{}
	})
}

func thinkingTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return thinkingTickMsg{}
	})
}
