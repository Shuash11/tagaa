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
		parts := []geminiPart{{Text: userMsg}}
		if system != "" {
			parts = append([]geminiPart{{Text: system + "\n\n" + userMsg}}, parts...)
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
	for _, part := range parts {
		part = strings.Trim(part, "\"'(),.;:!?")
		if strings.Contains(part, "/") || strings.Contains(part, "\\") {
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

func (p *Pipeline) plansSummary() string {
	if len(p.plans) == 0 {
		return "(no plans available)"
	}
	var sb strings.Builder
	for i, pl := range p.plans {
		sb.WriteString(fmt.Sprintf("\n--- Plan %d: %s ---\n", i+1, pl.AgentName))
		sb.WriteString(pl.Plan)
		if len(pl.Plan) > 800 {
			sb.WriteString("\n[truncated]")
		}
	}
	return sb.String()
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
	if len(plans) > 0 {
		return plans[0].AgentName
	}
	return p.agents[0].Name
}
