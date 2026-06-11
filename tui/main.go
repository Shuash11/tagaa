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
	"groq":       "https://api.groq.com/openai",
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
		messages: []Message{
			{Kind: MsgSystem, Content: "◆ TAGAA v0.1.0 — Terminal Autonomous Group AI Assistant"},
			{Kind: MsgSystem, Content: "◆ No agents yet. Press Ctrl+S to add agents and configure API keys"},
			{Kind: MsgSystem, Content: "  Ctrl+B toggle sidebar"},
			{Kind: MsgSystem, Content: ""},
		},
		apiKeys:       make(map[string]string),
		models:        make(map[string][]string),
		modelsLoading: make(map[string]bool),
		modelErrors:   make(map[string]string),
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
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			msg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			return fetchModelsMsg{provider: id, err: fmt.Errorf(msg)}
		}
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
		m.modelsLoading[msg.provider] = false
		if msg.err != nil {
			m.modelErrors[msg.provider] = msg.err.Error()
			m.models[msg.provider] = nil
		} else {
			delete(m.modelErrors, msg.provider)
			m.models[msg.provider] = msg.models
		}
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
		case "up", "down", "left", "right":
			return m, nil
		case "enter":
			if t := strings.TrimSpace(m.input); t != "" {
				m.messages = append(m.messages, Message{Kind: MsgUser, Content: t})
				m.messages = append(m.messages, Message{Kind: MsgSystem, Content: ""})
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

func (m model) updSidebarConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.sidebarConfig = false
		return m, nil
	case "up":
		if m.sidebarStep == 0 {
			if m.sidebarSel > 0 {
				m.sidebarSel--
			}
		} else if m.sidebarStep == 1 {
			// find previous provider with API key
			for i := m.sidebarCur - 1; i >= 0; i-- {
				if m.apiKeys[providers[i].id] != "" {
					m.sidebarCur = i
					break
				}
			}
		} else if m.sidebarStep == 2 {
			if m.sidebarCur > 0 {
				m.sidebarCur--
			}
		}
		return m, nil
	case "down":
		if m.sidebarStep == 0 {
			if m.sidebarSel < len(m.agents)-1 {
				m.sidebarSel++
			}
		} else if m.sidebarStep == 1 {
			// find next provider with API key
			for i := m.sidebarCur + 1; i < len(providers); i++ {
				if m.apiKeys[providers[i].id] != "" {
					m.sidebarCur = i
					break
				}
			}
		} else if m.sidebarStep == 2 {
			models := m.models[m.sidebarProv]
			if m.sidebarCur < len(models)-1 {
				m.sidebarCur++
			}
		}
		return m, nil
	case "enter":
		if m.sidebarStep == 0 {
			// start provider selection, find first provider with API key
			m.sidebarCur = 0
			hasKey := false
			for _, p := range providers {
				if m.apiKeys[p.id] != "" {
					hasKey = true
					break
				}
			}
			if !hasKey {
				m.sidebarConfig = false
				return m, nil
			}
			for i, p := range providers {
				if m.apiKeys[p.id] != "" {
					m.sidebarCur = i
					break
				}
			}
			m.sidebarStep = 1
			m.sidebarProv = ""
			return m, nil
		} else if m.sidebarStep == 1 {
			// provider selected, start model selection
			pid := providers[m.sidebarCur].id
			m.sidebarProv = pid
			m.agents[m.sidebarSel].provider = pid
			m.agents[m.sidebarSel].model = ""
			models := m.models[pid]
			if len(models) > 0 {
				m.sidebarStep = 2
				m.sidebarCur = 0
			} else {
				m.sidebarConfig = false
			}
			return m, nil
		} else if m.sidebarStep == 2 {
			// model selected
			models := m.models[m.sidebarProv]
			if m.sidebarCur < len(models) {
				m.agents[m.sidebarSel].model = models[m.sidebarCur]
			}
			m.sidebarConfig = false
			return m, nil
		}
	}
	return m, nil
}

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
			if key != "" {
				m.modelsLoading[id] = true
				delete(m.modelErrors, id)
				return m, fetchModelsCmd(id, key)
			}
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
					m.agents[m.agentCur].name = strings.TrimSpace(m.agentTemp)
				}
				m.agentField = 1
				m.agentTemp = m.agents[m.agentCur].provider
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
					m.agents[m.agentCur].provider = m.agentTemp
					m.agents[m.agentCur].model = ""
				}
				m.agentField = 2
				m.agentTemp = m.agents[m.agentCur].model
			case "esc":
				m.agentEdit = false
			case "up":
				idx := len(providers) - 1
				for i, p := range providers {
					if p.id == m.agentTemp && i > 0 {
						idx = i - 1
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
		case 2:
			models := m.models[m.agents[m.agentCur].provider]
			switch msg.String() {
			case "enter":
				if m.agentTemp != "" {
					m.agents[m.agentCur].model = m.agentTemp
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
			m.agentTemp = m.agents[m.agentCur].name
		}
	case "a":
		m.agents = append(m.agents, agentCfg{
			name:    fmt.Sprintf("Agent %d", len(m.agents)+1),
			enabled: true,
		})
		m.agentCur = len(m.agents) - 1
	case "d":
		if len(m.agents) == 0 {
			return m, nil
		}
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

func renderMessage(msg Message, width int) string {
	switch msg.Kind {
	case MsgUser:
		return lipgloss.NewStyle().Bold(true).Foreground(accentC).Render("  You: ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#E6E6E6")).Render(msg.Content)

	case MsgAgent:
		badge := lipgloss.NewStyle().Foreground(msg.Color).Bold(true).Render("◆ " + msg.AgentName)
		body := lipgloss.NewStyle().PaddingLeft(5).Width(width - 2).Render(msg.Content)
		return badge + "\n" + body

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

	hdr := lipgloss.NewStyle().
		Background(bg).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderC).
		Width(mw - 2).
		PaddingLeft(1).
		Render(lipgloss.NewStyle().Background(bg).Render(
			lipgloss.NewStyle().Bold(true).Foreground(accentC).Render(" TAGAA") +
				phaseStr +
				lipgloss.NewStyle().Faint(true).Render("  v0.1.0"),
		))

	inpWidth := mw - 4
	cursor := lipgloss.NewStyle().Background(lipgloss.Color("#E6E6E6")).Render(" ")
	inpContent := lipgloss.NewStyle().Bold(true).Foreground(accentC).Render("λ ") +
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E6E6E6")).Render(m.input) +
		cursor
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

	// sidebar config dropdown overlay
	if m.sidebarConfig && m.sidebarStep > 0 {
		msgs = m.sidebarDropdown(msgW, msgH)
	}

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

func (m model) sidebarDropdown(w, h int) string {
	dw := 40
	if w < dw+4 { dw = w - 4 }
	if dw < 20 { dw = 20 }

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
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render(" for " + agent.name))
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
		b.WriteString(lipgloss.NewStyle().Faint(true).Foreground(muteC).Render(" for " + agent.name))
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
			pName := a.provider
			if pName == "" || a.model == "" {
				pName = "no key"
			}
			prefix := "  "
			if m.sidebarConfig && m.sidebarStep == 0 && i == m.sidebarSel {
				prefix = lipgloss.NewStyle().Foreground(accentC).Render("▸ ")
			} else {
				prefix = "  "
			}
			pad(fmt.Sprintf("%s%s", prefix, a.name))
			pad(lipgloss.NewStyle().Foreground(muteC).Width(18).Render(fmt.Sprintf("    %s", pName)))
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

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
