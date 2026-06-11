package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	bg      = lipgloss.Color("#16161e")
	accentC = lipgloss.Color("#00CED1")
	blueC   = lipgloss.Color("#5B8DEF")
	greenC  = lipgloss.Color("#4CCD6B")
	redC    = lipgloss.Color("#E06C75")
	muteC   = lipgloss.Color("#5c6370")
	borderC = lipgloss.Color("#2c313a")
	sbarBg  = lipgloss.Color("#16161e")
	dialogBg = lipgloss.Color("#1a1a2e")
	tabBg   = lipgloss.Color("#0f0f1a")
)

type provider struct{ id, label string }

var providers = []provider{
	{"anthropic", "Anthropic"}, {"openai", "OpenAI"},
	{"gemini", "Gemini"}, {"mistral", "Mistral"},
	{"deepseek", "DeepSeek"}, {"xai", "xAI"}, {"nvidia", "NVIDIA"},
	{"groq", "Groq"}, {"together", "Together"},
	{"openrouter", "OpenRouter"}, {"cohere", "Cohere"},
}

var baseURLs = map[string]string{
	"anthropic":  "https://api.anthropic.com",
	"openai":     "https://api.openai.com",
	"gemini":     "https://generativelanguage.googleapis.com",
	"mistral":    "https://api.mistral.ai",
	"deepseek":   "https://api.deepseek.com",
	"xai":        "https://api.x.ai",
	"nvidia":     "https://api.nvcf.nvidia.com",
	"groq":       "https://api.groq.com",
	"together":   "https://api.together.xyz",
	"openrouter": "https://openrouter.ai/api",
	"cohere":     "https://api.cohere.com",
}

type agentCfg struct {
	name     string
	provider string
	model    string
	enabled  bool
}

type model struct {
	agents        []agentCfg
	messages      []string
	input         string
	apiKeys       map[string]string
	models        map[string][]string
	modelsLoading map[string]bool
	sidebar       bool
	settings      bool
	settingsTab   int
	setCur        int
	setEdit       bool
	setKey        string
	modeSelect    bool
	selModel      string
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

func initialModel() model {
	return model{
		agents:        []agentCfg{},
		messages: []string{
			"  ◆ TAGAA v0.1.0 — Terminal Autonomous Group AI Assistant",
			"  ◆ No agents yet. Press Ctrl+S to add agents and configure API keys",
			"    Ctrl+B toggle sidebar",
			"",
		},
		apiKeys:       make(map[string]string),
		models:        make(map[string][]string),
		modelsLoading: make(map[string]bool),
		sidebar:       true,
	}
}

func fetchModelsCmd(id, key string) tea.Cmd {
	base, ok := baseURLs[id]
	if !ok || key == "" {
		return nil
	}
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", base+"/v1/models", nil)
		if err != nil {
			return fetchModelsMsg{provider: id, err: err}
		}
		switch id {
		case "anthropic":
			req.Header.Set("x-api-key", key)
			req.Header.Set("anthropic-version", "2023-06-01")
		case "gemini":
			q := req.URL.Query()
			q.Set("key", key)
			req.URL.RawQuery = q.Encode()
		default:
			req.Header.Set("Authorization", "Bearer "+key)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fetchModelsMsg{provider: id, err: err}
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fetchModelsMsg{provider: id, err: err}
		}
		var names []string
		var openAI struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &openAI); err == nil && len(openAI.Data) > 0 {
			for _, d := range openAI.Data {
				names = append(names, d.ID)
			}
			return fetchModelsMsg{provider: id, models: names}
		}
		var geminiResp struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.Unmarshal(body, &geminiResp); err == nil && len(geminiResp.Models) > 0 {
			for _, m := range geminiResp.Models {
				names = append(names, strings.TrimPrefix(m.Name, "models/"))
			}
			return fetchModelsMsg{provider: id, models: names}
		}
		return fetchModelsMsg{provider: id, err: fmt.Errorf("unrecognized response")}
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.ready = true
		return m, nil

	case fetchModelsMsg:
		if msg.err != nil {
			m.models[msg.provider] = nil
		} else {
			m.models[msg.provider] = msg.models
		}
		m.modelsLoading[msg.provider] = false
		return m, nil

	case tea.KeyMsg:
		if m.settings {
			return m.updSettings(msg)
		}
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
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
			m.modeSelect = false
			m.agentCur = 0
			m.agentEdit = false
			return m, nil
		case "up", "down", "left", "right":
			return m, nil
		case "enter":
			if t := strings.TrimSpace(m.input); t != "" {
				m.messages = append(m.messages, "  You: "+t)
				m.messages = append(m.messages, "")
				m.input = ""
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

func (m model) updSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "tab" || msg.String() == "shift+tab" {
		m.settingsTab = 1 - m.settingsTab
		m.setCur = 0
		m.setEdit = false
		m.modeSelect = false
		m.agentEdit = false
		return m, nil
	}
	if m.settingsTab == 0 {
		return m.updKeysTab(msg)
	}
	return m.updAgentTab(msg)
}

func (m model) updKeysTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.modeSelect {
		models := m.models[providers[m.setCur].id]
		switch msg.String() {
		case "escape":
			m.modeSelect = false
		case "enter":
			if m.selModel != "" {
				m.models[providers[m.setCur].id] = models
			}
			m.modeSelect = false
		case "up":
			idx := -1
			for i, n := range models {
				if n == m.selModel {
					idx = i; break
				}
			}
			if idx > 0 {
				m.selModel = models[idx-1]
			}
		case "down":
			idx := -1
			for i, n := range models {
				if n == m.selModel {
					idx = i; break
				}
			}
			if idx < len(models)-1 {
				m.selModel = models[idx+1]
			}
		}
		return m, nil
	}

	if m.setEdit {
		switch msg.String() {
		case "enter":
			id := providers[m.setCur].id
			key := strings.TrimSpace(m.setKey)
			m.apiKeys[id] = key
			m.setEdit = false
			m.setKey = ""
			if key != "" {
				m.modelsLoading[id] = true
				return m, fetchModelsCmd(id, key)
			}
			return m, nil
		case "escape":
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
	case "escape":
		m.settings = false
		m.modeSelect = false
	case "enter":
		id := providers[m.setCur].id
		if m.apiKeys[id] != "" && len(m.models[id]) > 0 {
			m.modeSelect = true
			m.selModel = ""
		} else {
			m.setEdit = true
			m.setKey = m.apiKeys[id]
		}
	case "up":
		if m.setCur > 0 {
			m.setCur--
		}
	case "down":
		if m.setCur < len(providers)-1 {
			m.setCur++
		}
	}
	return m, nil
}

func (m model) updAgentTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// if editing an agent field
	if m.agentEdit {
		switch m.agentField {
		case 0: // editing name
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.agentTemp) != "" {
					m.agents[m.agentCur].name = strings.TrimSpace(m.agentTemp)
				}
				m.agentField = 1
				m.agentTemp = m.agents[m.agentCur].provider
			case "escape":
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
		case 1: // editing provider
			switch msg.String() {
			case "enter":
				if m.agentTemp != "" {
					m.agents[m.agentCur].provider = m.agentTemp
					m.agents[m.agentCur].model = ""
				}
				m.agentField = 2
				m.agentTemp = m.agents[m.agentCur].model
				// skip model step if no provider set
				if m.agentTemp == "" {
					m.agentEdit = false
				}
			case "escape":
				m.agentEdit = false
			case "up":
				idx := len(providers) - 1
				for i, p := range providers {
					if p.id == m.agentTemp && i > 0 {
						idx = i - 1
						break
					}
					if p.id == m.agentTemp {
						idx = len(providers) - 1
						break
					}
				}
				m.agentTemp = providers[idx].id
			case "down":
				idx := 0
				for i, p := range providers {
					if p.id == m.agentTemp && i < len(providers)-1 {
						idx = i + 1
						break
					}
				}
				m.agentTemp = providers[idx].id
			}
		case 2: // editing model
			models := m.models[m.agents[m.agentCur].provider]
			switch msg.String() {
			case "enter":
				if m.agentTemp != "" {
					m.agents[m.agentCur].model = m.agentTemp
				}
				m.agentEdit = false
			case "escape":
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
						idx = i; break
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
						idx = i; break
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
	case "escape":
		m.settings = false
	case "enter":
		if m.agentCur >= 0 && m.agentCur < len(m.agents) {
			m.agentEdit = true
			m.agentField = 0
			m.agentTemp = m.agents[m.agentCur].name
		}
	case "a":
		m.agents = append(m.agents, agentCfg{
			name: fmt.Sprintf("Agent %d", len(m.agents)+1),
			enabled: true,
		})
		m.agentCur = len(m.agents) - 1
	case "d":
			m.agents = append(m.agents[:m.agentCur], m.agents[m.agentCur+1:]...)
			if m.agentCur >= len(m.agents) {
				m.agentCur = len(m.agents) - 1
			}
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

	hdr := lipgloss.NewStyle().
		Background(bg).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderC).
		Width(mw - 2).
		PaddingLeft(1).
		Render(lipgloss.NewStyle().Background(bg).Render(
			lipgloss.NewStyle().Bold(true).Foreground(accentC).Render(" TAGAA  ") +
				lipgloss.NewStyle().Faint(true).Render("Terminal Autonomous Group AI Assistant  ") +
				lipgloss.NewStyle().Faint(true).Render("v0.1.0"),
		))

	inpWidth := mw - 4
	inpContent := lipgloss.NewStyle().Background(bg).Render(
		lipgloss.NewStyle().Bold(true).Foreground(accentC).Render("λ ") +
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E6E6E6")).Render(m.input),
	)
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

	var body strings.Builder
	start := 0
	if len(m.messages) > msgH {
		start = len(m.messages) - msgH
	}
	for i := start; i < len(m.messages); i++ {
		line := m.messages[i]
		line += strings.Repeat(" ", max(0, msgW-len(line)))
		body.WriteString(line + "\n")
	}
	extra := msgH - (len(m.messages) - start)
	for range extra {
		body.WriteString(strings.Repeat(" ", msgW) + "\n")
	}

	msgs := lipgloss.NewStyle().Background(bg).Width(msgW).Render(body.String())
	mainCol := lipgloss.JoinVertical(lipgloss.Top, hdr, msgs, inp)

	if m.sidebar {
		return lipgloss.JoinHorizontal(lipgloss.Top, mainCol, m.sideView())
	}
	return mainCol
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

	cw := dw - 2

	var b strings.Builder
	b.WriteString(m.tabBar())
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(blueC).Render("API Keys & Models"))
	b.WriteString("\n\n")

	for i, p := range providers {
		cursor := "  "
		color := lipgloss.Color("#E6E6E6")
		if i == m.setCur && !m.setEdit && !m.modeSelect {
			cursor = "▸ "
			color = blueC
		}

		key := m.apiKeys[p.id]
		status := "○ empty"
		if key != "" {
			masked := strings.Repeat("●", max(1, len(key)))
			if len(masked) > 12 {
				masked = masked[:12]
			}
			status = masked
		}
		if m.modelsLoading[p.id] {
			status = "⟳ loading"
		}

		b.WriteString(lipgloss.NewStyle().Foreground(color).Render(
			fmt.Sprintf("%s%-12s  %s", cursor, p.label, status),
		))
		b.WriteString("\n")

		if i == m.setCur && m.apiKeys[p.id] != "" && !m.setEdit {
			models := m.models[p.id]
			if len(models) > 0 && !m.modelsLoading[p.id] {
				for _, mn := range models {
					sel := "  "
					modColor := muteC
					if m.modeSelect {
						if mn == m.selModel {
							sel = "▸ "
							modColor = greenC
						}
					}
					line := fmt.Sprintf("   %s%s", sel, mn)
					if len(line) > cw {
						line = line[:cw]
					}
					b.WriteString(lipgloss.NewStyle().Foreground(modColor).Render(line))
					b.WriteString("\n")
				}
			} else if m.modelsLoading[p.id] {
				b.WriteString(lipgloss.NewStyle().Foreground(muteC).Render("   ⟳ Fetching models..."))
				b.WriteString("\n")
			} else if key != "" {
				b.WriteString(lipgloss.NewStyle().Foreground(muteC).Render("   Press Enter to fetch models"))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	if m.modeSelect {
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render("↑↓ choose model, Enter confirm, Esc back"))
	} else if m.setEdit {
		b.WriteString("Enter API key, Esc to cancel")
	} else {
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render("↑↓ provider · Enter edit · Tab: Agents"))
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

	// header
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

		pName := a.provider
		if pName == "" {
			pName = "(none)"
		}
		mod := a.model
		if mod == "" {
			mod = "(none)"
		}

		// if editing this agent
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
				line := fmt.Sprintf("%s%-16s %-14s %s", cursor, a.name, selProv, mod)
				b.WriteString(lipgloss.NewStyle().Foreground(greenC).Render(line))
			case 2:
				line := fmt.Sprintf("%s%-16s %-14s %s", cursor, a.name, pName, " ▸ "+m.agentTemp)
				b.WriteString(lipgloss.NewStyle().Foreground(greenC).Render(line))
			}
		} else {
			line := fmt.Sprintf("%s%-16s %-14s %s", cursor, a.name, pName, mod)
			b.WriteString(lipgloss.NewStyle().Foreground(color).Render(line))
		}
		b.WriteString("\n")

		// show provider's models below if editing provider/model
		if m.agentEdit && i == m.agentCur {
			if m.agentField == 1 {
				// show provider list
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
				models := m.models[m.agents[m.agentCur].provider]
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

func (m model) sideView() string {
	var b strings.Builder

	pad := func(s string) {
		b.WriteString(lipgloss.NewStyle().Background(sbarBg).Width(20).Render(s))
		b.WriteString("\n")
	}
	pad(lipgloss.NewStyle().Bold(true).Foreground(accentC).Render(" TAGAA"))
	pad("")
	pad(lipgloss.NewStyle().Faint(true).Render(" STATUS"))
	pad(fmt.Sprintf(" %s Idle",
		lipgloss.NewStyle().Foreground(greenC).Render("●"),
	))
	pad("")
	pad(lipgloss.NewStyle().Faint(true).Render(" AGENTS"))
	if len(m.agents) == 0 {
		pad(lipgloss.NewStyle().Foreground(muteC).Render("  (none configured)"))
	} else {
		for _, a := range m.agents {
			mod := a.model
			if mod == "" {
				mod = "(no model)"
			}
			pad(fmt.Sprintf("  %s", a.name))
			pad(lipgloss.NewStyle().Foreground(muteC).Width(18).Render(fmt.Sprintf("    %s", mod)))
		}
	}
	pad("")
	pad(lipgloss.NewStyle().Faint(true).Render(" KEYS"))
	pad(lipgloss.NewStyle().Foreground(blueC).Render(" Ctrl+S Setup"))
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

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
