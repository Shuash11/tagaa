package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) updSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "tab" || msg.String() == "shift+tab" {
		m.settingsTab = 1 - m.settingsTab
		m.setCur = 0
		m.setEdit = false
		m.agentEdit = false
		return m, nil
	}
	if m.settingsTab == 0 {
		return m.updKeysTab(msg)
	}
	return m.updAgentTab(msg)
}

func (m model) updKeysTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.setEdit {
		switch msg.String() {
		case "enter":
			id := providers[m.setCur].id
			key := strings.TrimSpace(m.setKey)
			m.apiKeys[id] = key
			m.setEdit = false
			m.setKey = ""
			saveConfig(m)
			if key != "" {
				m.modelsLoading[id] = true
				delete(m.modelErrors, id)
				return m, fetchModelsCmd(id, key)
			}
			delete(m.modelErrors, id)
			delete(m.models, id)
			return m, nil
		case "esc":
			m.setEdit = false
			m.setKey = ""
			return m, nil
		case "backspace":
			if len(m.setKey) > 0 {
				m.setKey = m.setKey[:len(m.setKey)-1]
			}
			return m, nil
		default:
			if !strings.HasPrefix(msg.String(), "ctrl") {
				s := msg.String()
				if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
					m.setKey += s
				}
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "esc":
		m.settings = false
	case "enter":
		m.setEdit = true
		m.setKey = m.apiKeys[providers[m.setCur].id]
	case "up":
		if m.setCur > 0 {
			m.setCur--
		}
	case "down":
		if m.setCur < len(providers)-1 {
			m.setCur++
		}
	case "d":
		id := providers[m.setCur].id
		m.apiKeys[id] = ""
		delete(m.modelErrors, id)
		m.modelsLoading[id] = false
		delete(m.models, id)
		saveConfig(m)
	}
	return m, nil
}

func (m model) updAgentTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.agentEdit {
		switch m.agentField {
		case 0:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.agentTemp) != "" {
					m.agents[m.agentCur].Name = strings.TrimSpace(m.agentTemp)
				}
				m.agentField = 1
				m.agentTemp = m.agents[m.agentCur].Provider
				saveConfig(m)
			case "esc":
				m.agentEdit = false
			case "backspace":
				if len(m.agentTemp) > 0 {
					m.agentTemp = m.agentTemp[:len(m.agentTemp)-1]
				}
			default:
				s := msg.String()
				if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
					m.agentTemp += s
				}
			}
		case 1:
			switch msg.String() {
			case "enter":
				if m.agentTemp != "" {
					m.agents[m.agentCur].Provider = m.agentTemp
					m.agents[m.agentCur].Model = ""
				}
				m.agentField = 2
				m.agentTemp = m.agents[m.agentCur].Model
				saveConfig(m)
			case "esc":
				m.agentEdit = false
			case "up":
				idx := -1
				for i, p := range providers {
					if p.id == m.agentTemp && i > 0 {
						idx = i - 1
						break
					}
				}
				if idx >= 0 {
					m.agentTemp = providers[idx].id
				}
			case "down":
				idx := -1
				for i, p := range providers {
					if p.id == m.agentTemp && i < len(providers)-1 {
						idx = i + 1
						break
					}
				}
				if idx >= 0 {
					m.agentTemp = providers[idx].id
				}
			}
		case 2:
			models := m.models[m.agents[m.agentCur].Provider]
			switch msg.String() {
			case "enter":
				if m.agentTemp != "" {
					m.agents[m.agentCur].Model = m.agentTemp
					saveConfig(m)
				}
				m.agentEdit = false
			case "esc":
				m.agentEdit = false
			case "up":
				if len(models) == 0 {
					return m, nil
				}
				if m.agentTemp == "" {
					m.agentTemp = models[0]
					return m, nil
				}
				idx := -1
				for i, n := range models {
					if n == m.agentTemp {
						idx = i
						break
					}
				}
				if idx <= 0 {
					m.agentTemp = models[len(models)-1]
				} else {
					m.agentTemp = models[idx-1]
				}
			case "down":
				if len(models) == 0 {
					return m, nil
				}
				if m.agentTemp == "" {
					m.agentTemp = models[0]
					return m, nil
				}
				idx := -1
				for i, n := range models {
					if n == m.agentTemp {
						idx = i
						break
					}
				}
				if idx < 0 || idx >= len(models)-1 {
					m.agentTemp = models[0]
				} else {
					m.agentTemp = models[idx+1]
				}
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.settings = false
	case "enter":
		if m.agentCur >= 0 && m.agentCur < len(m.agents) {
			m.agentEdit = true
			m.agentField = 0
			m.agentTemp = m.agents[m.agentCur].Name
		}
	case "a":
		m.agents = append(m.agents, agentCfg{
			Name:    fmt.Sprintf("Agent %d", len(m.agents)+1),
			Enabled: true,
		})
		m.agentCur = len(m.agents) - 1
		saveConfig(m)
	case "d":
		if len(m.agents) == 0 {
			return m, nil
		}
		m.agents = append(m.agents[:m.agentCur], m.agents[m.agentCur+1:]...)
		if m.agentCur >= len(m.agents) {
			m.agentCur = len(m.agents) - 1
		}
		saveConfig(m)
	case "up":
		if m.agentCur > 0 {
			m.agentCur--
		}
	case "down":
		if m.agentCur < len(m.agents)-1 {
			m.agentCur++
		}
	}
	return m, nil
}

func (m model) tabBar() string {
	tabs := []string{"  API Keys  ", "  Agents  "}
	var parts []string
	for i, t := range tabs {
		s := lipgloss.NewStyle().Padding(0, 1)
		if i == m.settingsTab {
			s = s.Bold(true).Foreground(accentC).Background(tabBg)
		} else {
			s = s.Foreground(muteC)
		}
		parts = append(parts, s.Render(t))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m model) keysView(msgW, msgH int) string {
	dw := 56
	if msgW-4 < dw {
		dw = msgW - 4
	}
	if dw < 30 {
		dw = 30
	}

	dialogStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(blueC).
		Padding(1, 1).
		Width(dw).
		Background(dialogBg)

	var b strings.Builder
	b.WriteString(m.tabBar())
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(blueC).Render("API Keys & Models"))
	b.WriteString("\n\n")

	for i, p := range providers {
		cursor := "  "
		color := lipgloss.Color("#E6E6E6")
		if i == m.setCur && !m.setEdit {
			cursor = "▸ "
			color = blueC
		}

		key := m.apiKeys[p.id]
		status := "○ empty"

		if m.setEdit && i == m.setCur {
			masked := strings.Repeat("●", len(m.setKey))
			if masked == "" {
				masked = "▋"
			}
			status = masked
		} else if m.modelsLoading[p.id] {
			status = "⟳ loading"
		} else if errMsg, ok := m.modelErrors[p.id]; ok {
			status = lipgloss.NewStyle().Foreground(redC).Render("✗ " + errMsg)
		} else if key != "" {
			masked := strings.Repeat("●", max(1, min(len(key), 12)))
			status = masked
		}

		b.WriteString(lipgloss.NewStyle().Foreground(color).Render(
			fmt.Sprintf("%s%-12s  %s", cursor, p.label, status),
		))
		b.WriteString("\n")

		if i == m.setCur && m.apiKeys[p.id] != "" && !m.setEdit {
			models := m.models[p.id]
			if len(models) > 0 && !m.modelsLoading[p.id] {
				for _, mn := range models {
					line := fmt.Sprintf("     %s", mn)
					if len(line) > dw-4 {
						line = line[:dw-4]
					}
					b.WriteString(lipgloss.NewStyle().Foreground(muteC).Render(line))
					b.WriteString("\n")
				}
			} else if m.modelsLoading[p.id] {
				b.WriteString(lipgloss.NewStyle().Foreground(muteC).Render("   ⟳ Fetching models..."))
				b.WriteString("\n")
			} else if _, ok := m.modelErrors[p.id]; ok {
				b.WriteString(lipgloss.NewStyle().Foreground(redC).Render("   ✗ API key rejected or unreachable"))
				b.WriteString("\n")
			} else if key != "" {
				b.WriteString(lipgloss.NewStyle().Foreground(muteC).Render("   Press Enter to fetch models"))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	if m.setEdit {
		b.WriteString("Enter API key, Esc to cancel")
	} else {
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render("↑↓ provider · Enter edit · d delete · Tab: Agents"))
	}

	dialog := dialogStyle.Render(b.String())
	return lipgloss.Place(msgW, msgH, lipgloss.Center, lipgloss.Center, dialog, lipgloss.WithWhitespaceBackground(bg))
}

func (m model) agentsView(msgW, msgH int) string {
	dw := 60
	if msgW-4 < dw {
		dw = msgW - 4
	}
	if dw < 36 {
		dw = 36
	}

	dialogStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(blueC).
		Padding(1, 1).
		Width(dw).
		Background(dialogBg)

	var b strings.Builder
	b.WriteString(m.tabBar())
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(blueC).Render("Agent Configuration"))
	b.WriteString("\n\n")

	if len(m.agents) > 0 {
		hdr := lipgloss.NewStyle().Faint(true).Foreground(muteC).Render(
			fmt.Sprintf("%-16s %-14s %s", "Name", "Provider", "Model"),
		)
		b.WriteString("  " + hdr)
		b.WriteString("\n")
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(muteC).Render("  No agents yet — press a to add one"))
		b.WriteString("\n")
	}

	for i, a := range m.agents {
		cursor := "  "
		color := lipgloss.Color("#E6E6E6")
		if i == m.agentCur && !m.agentEdit {
			cursor = "▸ "
			color = blueC
		}

		pName := a.Provider
		if pName == "" {
			pName = "(none)"
		}
		mod := a.Model
		if mod == "" {
			mod = "(none)"
		}

		if m.agentEdit && i == m.agentCur {
			switch m.agentField {
			case 0:
				name := m.agentTemp
				line := fmt.Sprintf("%s%-16s %-14s %s", cursor, name+"_", pName, mod)
				b.WriteString(lipgloss.NewStyle().Foreground(color).Render(line))
			case 1:
				pDisplay := m.agentTemp
				for _, pp := range providers {
					if pp.id == pDisplay {
						pDisplay = pp.label
						break
					}
				}
				selProv := " ▸ " + pDisplay
				line := fmt.Sprintf("%s%-16s %-14s %s", cursor, a.Name, selProv, mod)
				b.WriteString(lipgloss.NewStyle().Foreground(greenC).Render(line))
			case 2:
				line := fmt.Sprintf("%s%-16s %-14s %s", cursor, a.Name, pName, " ▸ "+m.agentTemp)
				b.WriteString(lipgloss.NewStyle().Foreground(greenC).Render(line))
			}
		} else {
			line := fmt.Sprintf("%s%-16s %-14s %s", cursor, a.Name, pName, mod)
			b.WriteString(lipgloss.NewStyle().Foreground(color).Render(line))
		}
		b.WriteString("\n")

		if m.agentEdit && i == m.agentCur {
			if m.agentField == 1 {
				for _, pp := range providers {
					sel := "  "
					pc := muteC
					if pp.id == m.agentTemp {
						sel = "▸ "
						pc = greenC
					}
					b.WriteString(lipgloss.NewStyle().Foreground(pc).Render(fmt.Sprintf("     %s%s", sel, pp.label)))
					b.WriteString("\n")
				}
			}
			if m.agentField == 2 {
				models := m.models[m.agents[m.agentCur].Provider]
				if len(models) > 0 {
					for _, mn := range models {
						sel := "  "
						mc := muteC
						if mn == m.agentTemp {
							sel = "▸ "
							mc = greenC
						}
						line := fmt.Sprintf("     %s%s", sel, mn)
						if len(line) > dw-6 {
							line = line[:dw-6]
						}
						b.WriteString(lipgloss.NewStyle().Foreground(mc).Render(line))
						b.WriteString("\n")
					}
				} else {
					b.WriteString(lipgloss.NewStyle().Foreground(muteC).Render("     No models loaded — set API key first"))
					b.WriteString("\n")
				}
			}
		}
	}

	b.WriteString("\n")
	if m.agentEdit {
		switch m.agentField {
		case 0:
			b.WriteString("Editing name — Enter confirm, Esc cancel")
		case 1:
			b.WriteString("↑↓ choose provider, Enter confirm, Esc cancel")
		case 2:
			b.WriteString("↑↓ choose model, Enter confirm, Esc cancel")
		}
	} else {
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render(
			"↑↓ select · Enter edit · [a]dd [d]elete · Tab: Keys",
		))
	}

	dialog := dialogStyle.Render(b.String())
	return lipgloss.Place(msgW, msgH, lipgloss.Center, lipgloss.Center, dialog, lipgloss.WithWhitespaceBackground(bg))
}
