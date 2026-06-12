package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderProgressBar(score, maxScore float64, width int) string {
	if maxScore <= 0 {
		maxScore = 1
	}
	frac := score / maxScore
	if frac > 1 {
		frac = 1
	}
	if frac < 0 {
		frac = 0
	}
	full := int(frac * float64(width))
	return strings.Repeat("█", full) + strings.Repeat("░", width-full)
}

func renderVoteBox(phase string, scores map[string]float64, winner string, entries []VoteEntry, boxWidth int) string {
	if boxWidth < 30 {
		boxWidth = 30
	}
	innerW := boxWidth - 2

	var b strings.Builder

	wl := func(s string) {
		if len(s) > innerW {
			s = s[:innerW]
		}
		b.WriteString("║" + s + strings.Repeat(" ", innerW-len(s)) + "║\n")
	}

	// Top border
	b.WriteString("╔" + strings.Repeat("═", innerW) + "╗\n")

	// Title
	title := "PLAN VOTE"
	winTag := "← WINNER"
	if strings.EqualFold(phase, "executor") {
		title = "EXECUTOR VOTE"
		winTag = "← SELECTED"
	}
	wl("  " + title)

	// Separator
	b.WriteString("╠" + strings.Repeat("═", innerW) + "╣\n")

	// Spacer
	wl("")

	// Score bars
	maxScore := 0.0
	for _, sc := range scores {
		if sc > maxScore {
			maxScore = sc
		}
	}

	barWidth := innerW - 24
	if barWidth < 8 {
		barWidth = 8
	}
	if barWidth > 16 {
		barWidth = 16
	}

	names := make([]string, 0, len(scores))
	for n := range scores {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		sc := scores[name]
		bar := renderProgressBar(sc, maxScore, barWidth)
		scoreStr := fmt.Sprintf("%.2f", sc)
		tag := ""
		if name == winner {
			tag = "  " + winTag
		}
		wl(fmt.Sprintf("  %s %s %s%s", name, bar, scoreStr, tag))
	}

	// Spacer
	wl("")

	// Voter section separator
	sepLabel := " WHO VOTED FOR WHAT "
	padTotal := innerW - len(sepLabel)
	if padTotal >= 4 {
		leftPad := padTotal / 2
		rightPad := padTotal - leftPad
		b.WriteString("╠" + strings.Repeat("═", leftPad) + sepLabel + strings.Repeat("═", rightPad) + "╣\n")
	} else {
		wl(sepLabel)
	}

	// Spacer
	wl("")

	// Vote entries
	for _, e := range entries {
		line := fmt.Sprintf("  %s ──→ %s", e.Voter, e.VotedFor)
		if e.Reason != "" {
			line += fmt.Sprintf("  %q", e.Reason)
		}
		wl(line)
	}

	// Spacer
	wl("")

	// Bottom border
	b.WriteString("╚" + strings.Repeat("═", innerW) + "╝")

	return b.String()
}

func (m model) renderMessage(msg Message, idx int, width int) string {
	switch msg.Kind {
	case MsgUser:
		lines := strings.Split(msg.Content, "\n")
		var b strings.Builder
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accentC).Render("  ─ You"))
		b.WriteString("\n")
		for _, line := range lines {
			b.WriteString(lipgloss.NewStyle().Foreground(accentC).Render("  │ "))
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#E6E6E6")).Render(line))
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
			barBody.WriteString(barStyle.Render("  │ "))
			barBody.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#E6E6E6")).Render(line))
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
		var result VoteResult
		if err := json.Unmarshal([]byte(msg.Content), &result); err == nil {
			return renderVoteBox(result.Phase, result.Scores, result.Winner, result.Entries, width)
		}
		return lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(blueC).
			PaddingLeft(1).
			Width(width-2).
			Render(
				lipgloss.NewStyle().Bold(true).Foreground(blueC).Render("VOTE RESULT")+"\n"+
					msg.Content,
			)

	case MsgPlan:
		if !m.planExpanded[idx] {
			var plan PlanSummary
			summary := ""
			if err := json.Unmarshal([]byte(msg.Content), &plan); err == nil {
				summary = plan.Summary
			}
			if summary == "" {
				lines := strings.Split(msg.Content, "\n")
				if len(lines) > 0 {
					summary = strings.TrimSpace(lines[0])
				}
			}
			if len(summary) > 60 {
				summary = summary[:60] + "..."
			}
			planCount := 0
			for j := 0; j <= idx; j++ {
				if m.messages[j].Kind == MsgPlan {
					planCount++
				}
			}
			collapsed := fmt.Sprintf("[ %s ] %s [expand: %d]", msg.AgentName, summary, planCount)
			return lipgloss.NewStyle().Foreground(msg.Color).Render(collapsed)
		}

		var plan PlanSummary
		if err := json.Unmarshal([]byte(msg.Content), &plan); err == nil {
			if strings.TrimSpace(plan.Summary) == "" && len(plan.Steps) == 0 {
				return ""
			}
			if strings.TrimSpace(plan.Summary) == "" && len(plan.Steps) > 0 {
				var sb strings.Builder
				sb.WriteString(lipgloss.NewStyle().Italic(true).Faint(true).Foreground(muteC).Render("(Agent provided no summary)"))
				for _, step := range plan.Steps {
					sb.WriteString("\n")
					sb.WriteString(step)
				}
				msg.Content = sb.String()
			}
		}

		boxWidth := width - 2
		if boxWidth < 30 {
			boxWidth = 30
		}
		innerW := boxWidth - 2

		title := " " + msg.AgentName + " "
		dashes := boxWidth - 2 - len(title)
		if dashes < 0 {
			dashes = 0
		}
		top := "┌─" + title + strings.Repeat("─", dashes) + "┐"
		bottom := "└" + strings.Repeat("─", boxWidth-2) + "┘"

		lines := strings.Split(msg.Content, "\n")
		var body strings.Builder
		for _, line := range lines {
			display := line
			if len(display) > innerW {
				display = display[:innerW]
			}
			padding := innerW - len(display)
			body.WriteString("│ ")
			body.WriteString(display)
			body.WriteString(strings.Repeat(" ", padding))
			body.WriteString(" │\n")
		}
		bodyStr := strings.TrimRight(body.String(), "\n")

		return lipgloss.NewStyle().Foreground(msg.Color).Render(top + "\n" + bodyStr + "\n" + bottom)

	case MsgPhaseDivider:
		txt := " " + msg.Content + " "
		pad := width - 2 - len(txt)
		if pad <= 0 {
			return lipgloss.NewStyle().Foreground(muteC).Render(msg.Content)
		}
		left := pad / 2
		right := pad - left
		line := strings.Repeat("─", left) + txt + strings.Repeat("─", right)
		return lipgloss.NewStyle().Foreground(muteC).Render(line)

	case MsgReview:
		var review ReviewResult
		if err := json.Unmarshal([]byte(msg.Content), &review); err == nil {
			var b strings.Builder
			title := fmt.Sprintf("[ %s — Reviewer ]", msg.AgentName)
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(msg.Color).Render(title))
			b.WriteString("\n")
			for _, line := range review.Lines {
				switch line.Type {
				case "pass":
					s := fmt.Sprintf("  ✓ %s:%d — %s", line.File, line.Line, line.Message)
					b.WriteString(lipgloss.NewStyle().Foreground(greenC).Render(s))
				default:
					s := fmt.Sprintf("  ⚠ %s:%d — %s", line.File, line.Line, line.Message)
					b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB000")).Render(s))
				}
				b.WriteString("\n")
			}
			b.WriteString(lipgloss.NewStyle().Bold(true).Render("Verdict: " + review.Verdict))
			return b.String()
		}
		return "[Review] " + msg.Content

	case MsgDissent:
		boxWidth := width - 2
		if boxWidth < 30 {
			boxWidth = 30
		}
		innerW := boxWidth - 2

		title := fmt.Sprintf(" %s — Acknowledging Dissent ", msg.AgentName)
		dashes := boxWidth - 2 - len(title)
		if dashes < 0 {
			dashes = 0
		}
		top := "┌─" + title + strings.Repeat("─", dashes) + "┐"
		bottom := "└" + strings.Repeat("─", boxWidth-2) + "┘"

		lines := strings.Split(msg.Content, "\n")
		var body strings.Builder
		for _, line := range lines {
			display := line
			if len(display) > innerW {
				display = display[:innerW]
			}
			padding := innerW - len(display)
			body.WriteString("│ ")
			body.WriteString(display)
			body.WriteString(strings.Repeat(" ", padding))
			body.WriteString(" │\n")
		}
		bodyStr := strings.TrimRight(body.String(), "\n")

		amber := lipgloss.Color("#FFB000")
		return lipgloss.NewStyle().Foreground(amber).Render(top + "\n" + bodyStr + "\n" + bottom)

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

	if m.showTokenEstimate {
		pad("")
		pad(lipgloss.NewStyle().Faint(true).Render(" USAGE"))
		pad(fmt.Sprintf(" ~%d tokens", m.totalTokens))
		cost := m.estimateCost()
		if cost > 0 {
			pad(fmt.Sprintf(" ~$%.3f est.", cost))
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

func (m model) estimateCost() float64 {
	if len(m.agents) == 0 || m.totalTokens == 0 {
		return 0
	}
	inputCost := 1.0
	outputCost := 5.0
	for _, a := range m.agents {
		if costs, ok := tokenCosts[a.Model]; ok {
			inputCost = costs.input
			outputCost = costs.output
			break
		}
	}
	estTokens := float64(m.totalTokens)
	return estTokens/2_000_000*inputCost + estTokens/2_000_000*outputCost
}
