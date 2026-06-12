package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func queryAgentSimple(ctx context.Context, agent agentCfg, apiKeys map[string]string, system, userMsg string) (string, error) {
	base, ok := baseURLs[agent.Provider]
	if !ok {
		return "", fmt.Errorf("unknown provider: %s", agent.Provider)
	}
	key := apiKeys[agent.Provider]
	if key == "" {
		return "", fmt.Errorf("no API key for %s", agent.Provider)
	}

	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
		}
		content, err := doQuery(ctx, agent, base, key, system, userMsg)
		if err == nil {
			return content, nil
		}
		lastErr = err
		if !isRetryable(err) {
			break
		}
	}
	return "", lastErr
}

func doQuery(ctx context.Context, agent agentCfg, base, key, system, userMsg string) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	var reqBody []byte
	var req *http.Request
	var err error

	switch agent.Provider {
	case "anthropic":
		type anthropicMsg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		type anthropicReq struct {
			Model     string         `json:"model"`
			System    string         `json:"system,omitempty"`
			Messages  []anthropicMsg `json:"messages"`
			MaxTokens int            `json:"max_tokens"`
		}
		reqBody, _ = json.Marshal(anthropicReq{
			Model: agent.Model, System: system,
			Messages:  []anthropicMsg{{Role: "user", Content: userMsg}},
			MaxTokens: 4096,
		})
		req, err = http.NewRequestWithContext(ctx, "POST", base+"/v1/messages", strings.NewReader(string(reqBody)))
		if err != nil {
			return "", err
		}
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("Content-Type", "application/json")

	case "gemini":
		type geminiPart struct {
			Text string `json:"text,omitempty"`
		}
		type geminiContent struct {
			Role  string        `json:"role,omitempty"`
			Parts []geminiPart  `json:"parts"`
		}
		type geminiReq struct {
			Contents []geminiContent `json:"contents"`
		}
		var parts []geminiPart
		if system != "" {
			parts = []geminiPart{{Text: system + "\n\n" + userMsg}}
		} else {
			parts = []geminiPart{{Text: userMsg}}
		}
		reqBody, _ = json.Marshal(geminiReq{Contents: []geminiContent{{Parts: parts}}})
		url := fmt.Sprintf("%s/v1/models/%s:generateContent?key=%s", base, agent.Model, key)
		req, err = http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")

	default:
		type openAIMsg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		type openAIReq struct {
			Model     string       `json:"model"`
			Messages  []openAIMsg  `json:"messages"`
			MaxTokens int          `json:"max_tokens,omitempty"`
		}
		msgs := []openAIMsg{}
		if system != "" {
			msgs = append(msgs, openAIMsg{Role: "system", Content: system})
		}
		msgs = append(msgs, openAIMsg{Role: "user", Content: userMsg})
		reqBody, _ = json.Marshal(openAIReq{Model: agent.Model, Messages: msgs, MaxTokens: 4096})
		req, err = http.NewRequestWithContext(ctx, "POST", base+"/v1/chat/completions", strings.NewReader(string(reqBody)))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	content := extractFinalText(agent.Provider, body)
	if content == "" {
		return "", fmt.Errorf("empty response")
	}
	return content, nil
}

func extractFilePaths(s string) []string {
	var files []string
	seen := make(map[string]bool)
	parts := strings.Fields(s)
	extPatterns := []string{".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".java", ".c", ".cpp", ".h", ".css", ".yaml", ".yml", ".json", ".md", ".toml", ".txt", ".sh", ".bat", ".ps1", ".xml", ".html", ".htm", ".csv"}
	rootPatterns := []string{"src/", "tui/", "lib/", "test/", "tests/", "docs/", "config/", "pkg/", "cmd/", "internal/", ".github/", "dist/", "build/", "public/", "static/", "assets/"}
	for _, part := range parts {
		part = strings.Trim(part, "\"'(),.;:!?")
		hasSep := strings.Contains(part, "/") || strings.Contains(part, "\\")
		if !hasSep {
			continue
		}
		lower := strings.ToLower(part)
		hasExt := false
		for _, ext := range extPatterns {
			if strings.HasSuffix(lower, ext) {
				hasExt = true
				break
			}
		}
		hasRoot := false
		for _, root := range rootPatterns {
			if strings.HasPrefix(lower, root) {
				hasRoot = true
				break
			}
		}
		if hasExt || hasRoot {
			if !seen[part] {
				files = append(files, part)
				seen[part] = true
			}
		}
	}
	return files
}

func (p *Pipeline) agentColor(name string) lipgloss.Color {
	for i, a := range p.agents {
		if a.Name == name {
			return agentColors[i%len(agentColors)]
		}
	}
	return agentColors[0]
}

const maxSummaryTotal = 2000

func (p *Pipeline) plansSummary() string {
	if len(p.plans) == 0 {
		return "(no plans available)"
	}
	perPlan := maxSummaryTotal / len(p.plans)
	if perPlan < 100 {
		perPlan = 100
	}
	var sb strings.Builder
	for i, pl := range p.plans {
		sb.WriteString(fmt.Sprintf("\n--- Plan %d: %s ---\n", i+1, pl.AgentName))
		runes := []rune(pl.Plan)
		if len(runes) > perPlan {
			sb.WriteString(string(runes[:perPlan]) + "\n[truncated]")
		} else {
			sb.WriteString(pl.Plan)
		}
	}
	result := sb.String()
	resultRunes := []rune(result)
	if len(resultRunes) > maxSummaryTotal {
		result = string(resultRunes[:maxSummaryTotal]) + "\n[total truncated]"
	}
	return result
}

func (p *Pipeline) parseVote(text string, plans []PlanResponse) string {
	re := regexp.MustCompile(`(?i)VOTE:\s*(\S+)`)
	m := re.FindStringSubmatch(text)
	if len(m) >= 2 {
		v := strings.TrimRight(m[1], ".,;!?")
		for _, pl := range plans {
			if strings.EqualFold(pl.AgentName, v) {
				return pl.AgentName
			}
		}
	}
	for _, pl := range plans {
		if strings.Contains(strings.ToLower(text), strings.ToLower(pl.AgentName)) {
			return pl.AgentName
		}
	}
	return ""
}

func extractReasonExcerpt(text string) string {
	text = strings.TrimSpace(text)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "VOTE:") || strings.HasPrefix(line, "```") {
			continue
		}
		runes := []rune(line)
		if len(runes) > 80 {
			return string(runes[:80]) + "..."
		}
		return line
	}
	return ""
}

func isRetryable(err error) bool {
	s := err.Error()
	return strings.Contains(s, "429") ||
		strings.Contains(s, "502") ||
		strings.Contains(s, "503") ||
		strings.Contains(s, "504") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "connection")
}

func (p *Pipeline) conversationContext(maxExchanges int) string {
	if len(p.history) == 0 || maxExchanges <= 0 {
		return ""
	}
	var sb strings.Builder
	count := 0
	for i := len(p.history) - 1; i >= 0 && count < maxExchanges; i-- {
		msg := p.history[i]
		if msg.Kind == MsgUser || msg.Kind == MsgAgent {
			prefix := "User"
			if msg.Kind == MsgAgent {
				prefix = msg.AgentName
			}
			content := msg.Content
			runes := []rune(content)
			if len(runes) > 200 {
				content = string(runes[:200]) + "..."
			}
			sb.WriteString(prefix + ": " + content + "\n")
			count++
		}
	}
	result := sb.String()
	if result != "" {
		return "\n\nRecent conversation:\n" + result
	}
	return ""
}
