package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderMessage(msg Message, width int) string {
	switch msg.Kind {
	case MsgUser:
		lines := strings.Split(msg.Content, "\n")
		var b strings.Builder
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentC).Render("  ─ You"))
		b.WriteString("\n")
		for _, line := range lines {
			trunc := line
			if len(trunc) > width-10 {
				trunc = trunc[:width-10]
			}
			b.WriteString(lipgloss.NewStyle().Foreground(accentC).Render("  │ "))
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#E6E6E6")).Render(trunc))
			b.WriteString("\n")
		}
		return strings.TrimRight(b.String(), "\n")

	case MsgAgent:
		barClr := msg.Color
		barStyle := lipgloss.NewStyle().Foreground(barClr)
		badge := barStyle.Bold(true).Render("  ◆ " + msg.AgentName)
		lines := strings.Split(msg.Content, "\n")
		var barBody strings.Builder
		for _, line := range lines {
			trunc := line
			if len(trunc) > width-10 {
				trunc = trunc[:width-10]
			}
			barBody.WriteString(barStyle.Render("  │ "))
			barBody.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#E6E6E6")).Render(trunc))
			barBody.WriteString("\n")
		}
		return badge + "\n" + strings.TrimRight(barBody.String(), "\n")

	case MsgSystem:
		return lipgloss.NewStyle().Foreground(muteC).Render("  " + msg.Content)

	case MsgError:
		return lipgloss.NewStyle().Foreground(redC).Render("  ✗ " + msg.Content)

	case MsgSuccess:
		return lipgloss.NewStyle().Foreground(greenC).Render("  ✓ " + msg.Content)

	case MsgVote:
		return lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(blueC).
			PaddingLeft(1).
			Width(width - 2).
			Render(
				lipgloss.NewStyle().Bold(true).Foreground(blueC).Render("VOTE RESULT") + "\n" +
					msg.Content,
			)

	default:
		return "  " + msg.Content
	}
}

func (m model) sidebarDropdown(w, h int) string {
	dw := 40
	if w < dw+4 {
		dw = w - 4
	}
	if dw < 20 {
		dw = 20
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentC).
		Padding(0, 1).
		Width(dw).
		Background(dialogBg)

	var b strings.Builder

	agent := m.agents[m.sidebarSel]
	if m.sidebarStep == 1 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentC).Render("Select Provider"))
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render(" for " + agent.Name))
		b.WriteString("\n\n")
		for i, p := range providers {
			if m.apiKeys[p.id] == "" {
				continue
			}
			sel := "  "
			color := lipgloss.Color("#E6E6E6")
			if i == m.sidebarCur {
				sel = "▸ "
				color = accentC
			}
			line := fmt.Sprintf("%s%s", sel, p.label)
			b.WriteString(lipgloss.NewStyle().Foreground(color).Render(line))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render("↑↓ Enter Esc"))
	} else if m.sidebarStep == 2 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentC).Render("Select Model"))
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render(" for " + agent.Name))
		b.WriteString("\n\n")
		models := m.models[m.sidebarProv]
		if len(models) > 0 {
			for i, mn := range models {
				sel := "  "
				color := lipgloss.Color("#E6E6E6")
				if i == m.sidebarCur {
					sel = "▸ "
					color = accentC
				}
				line := fmt.Sprintf("%s%s", sel, mn)
				if len(line) > dw {
					line = line[:dw]
				}
				b.WriteString(lipgloss.NewStyle().Foreground(color).Render(line))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(muteC).Render("  No models loaded"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render("↑↓ Enter Esc"))
	}

	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Center,
		style.Render(b.String()),
		lipgloss.WithWhitespaceBackground(bg),
	)
}

func (m model) sideView() string {
	var b strings.Builder

	pad := func(s string) {
		b.WriteString(lipgloss.NewStyle().Background(sbarBg).Width(20).Render(s))
		b.WriteString("\n")
	}
	pad(lipgloss.NewStyle().Bold(true).Foreground(accentC).Render(" TAGAA"))
	pad("")

	pad(lipgloss.NewStyle().Faint(true).Render(" STATUS"))
	statusDot := lipgloss.NewStyle().Foreground(muteC).Render("○")
	statusText := "Idle"
	if m.isRunning {
		statusDot = lipgloss.NewStyle().Foreground(greenC).Render("●")
		statusText = m.phase
	}
	pad(fmt.Sprintf(" %s %s", statusDot, statusText))

	pad("")
	pad(lipgloss.NewStyle().Faint(true).Render(" AGENTS"))
	if len(m.agents) == 0 {
		pad(lipgloss.NewStyle().Foreground(muteC).Render("  (none configured)"))
	} else {
		for i, a := range m.agents {
			pName := a.Provider
			if pName == "" {
				pName = "no provider"
			}
			modName := a.Model
			if modName == "" {
				modName = "no model"
			}
			prefix := "  "
			if m.sidebarConfig && m.sidebarStep == 0 && i == m.sidebarSel {
				prefix = lipgloss.NewStyle().Foreground(accentC).Render("▸ ")
			} else {
				prefix = "  "
			}
			pad(fmt.Sprintf("%s%s", prefix, a.Name))
			pad(lipgloss.NewStyle().Foreground(muteC).Width(18).Render(fmt.Sprintf("    %s", pName)))
			pad(lipgloss.NewStyle().Foreground(muteC).Width(18).Render(fmt.Sprintf("    %s", modName)))
		}
	}

	pad("")
	pad(lipgloss.NewStyle().Faint(true).Render(" KEYS"))
	pad(lipgloss.NewStyle().Foreground(blueC).Render(" Ctrl+S Setup"))
	if m.sidebarConfig {
		pad(lipgloss.NewStyle().Foreground(greenC).Render(" ● Config active"))
	} else {
		pad(lipgloss.NewStyle().Foreground(muteC).Render(" Ctrl+E Config"))
	}
	pad(lipgloss.NewStyle().Foreground(muteC).Render(" Ctrl+B Sidebar"))

	contentLines := strings.Count(b.String(), "\n")
	for i := contentLines; i < m.h; i++ {
		b.WriteString(lipgloss.NewStyle().Background(sbarBg).Width(20).Render(""))
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(22).
		Background(sbarBg).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(borderC).
		PaddingLeft(1).
		Render(b.String())
}

func (m model) cmdModeView(w, h int) string {
	dw := 50
	if w < dw+4 {
		dw = w - 4
	}
	if dw < 20 {
		dw = 20
	}
	dh := 15
	if h < dh+2 {
		dh = h - 2
	}
	if dh < 5 {
		dh = 5
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentC).
		Padding(0, 1).
		Width(dw).
		Background(dialogBg)

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentC).Render(" Sessions"))
	b.WriteString("\n\n")

	if len(m.cmdSessions) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(muteC).Render("  No saved sessions"))
	} else {
		for i, s := range m.cmdSessions {
			prefix := "  "
			clr := lipgloss.Color("#E6E6E6")
			if i == m.cmdCur {
				prefix = "▸ "
				clr = accentC
			}
			line := fmt.Sprintf("%s#%d  %s  (%d msgs)", prefix, s.ID, s.Timestamp, len(s.Messages))
			if len(line) > dw {
				line = line[:dw]
			}
			b.WriteString(lipgloss.NewStyle().Foreground(clr).Render(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render("  ↑↓ navigate · d delete · esc close"))

	return lipgloss.Place(w, h,
		lipgloss.Center, lipgloss.Center,
		style.Render(b.String()),
		lipgloss.WithWhitespaceBackground(bg),
	)
}
