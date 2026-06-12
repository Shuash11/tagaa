package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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
			short := err.Error()
			if len(short) > 40 {
				short = short[:40] + "…"
			}
			return fetchModelsMsg{provider: id, err: errors.New(short)}
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			short := fmt.Sprintf("HTTP %d", resp.StatusCode)
			if resp.StatusCode == 401 {
				short += " Invalid API key"
			} else if resp.StatusCode == 403 {
				short += " Access denied"
			} else if resp.StatusCode == 429 {
				short += " Rate limited"
			} else if resp.StatusCode >= 500 {
				short += " Server error"
			} else {
				short += " Error"
			}
			return fetchModelsMsg{provider: id, err: errors.New(short)}
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

func (m model) nextAgentName() string {
	n := len(m.agents)
	for i := 0; i < n; i++ {
		idx := (m.msgAgentIdx + i) % n
		a := m.agents[idx]
		if a.Enabled && a.Provider != "" && a.Model != "" && m.apiKeys[a.Provider] != "" {
			return a.Name
		}
	}
	return ""
}

func sendChatCmd(m model, ctx context.Context) tea.Cmd {
	var agent agentCfg
	found := false
	n := len(m.agents)
	for i := 0; i < n; i++ {
		idx := (m.msgAgentIdx + i) % n
		a := m.agents[idx]
		if a.Enabled && a.Provider != "" && a.Model != "" && m.apiKeys[a.Provider] != "" {
			agent = a
			found = true
			break
		}
	}
	if !found {
		msg := "No ready agent:"
		for _, a := range m.agents {
			why := ""
			if !a.Enabled {
				why = "disabled"
			} else if a.Provider == "" {
				why = "no provider"
			} else if a.Model == "" {
				why = "no model for " + a.Provider
			} else if m.apiKeys[a.Provider] == "" {
				why = "no API key for " + a.Provider
			} else {
				why = "unknown"
			}
			msg += " [" + a.Name + ": " + why + "]"
		}
		if len(msg) > 120 {
			msg = msg[:117] + "..."
		}
		return func() tea.Msg {
			return chatErrMsg{content: msg}
		}
	}

	base, ok := baseURLs[agent.Provider]
	if !ok {
		return func() tea.Msg {
			return chatErrMsg{content: "Unknown provider: " + agent.Provider}
		}
	}
	key := m.apiKeys[agent.Provider]

	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var systemParts []string
	var apiMsgs []chatMsg
	for _, msg := range m.messages {
		switch msg.Kind {
		case MsgSystem:
			if strings.TrimSpace(msg.Content) != "" {
				systemParts = append(systemParts, msg.Content)
			}
		case MsgUser:
			if strings.TrimSpace(msg.Content) != "" {
				apiMsgs = append(apiMsgs, chatMsg{Role: "user", Content: msg.Content})
			}
		case MsgAgent:
			if strings.TrimSpace(msg.Content) != "" {
				apiMsgs = append(apiMsgs, chatMsg{Role: "assistant", Content: msg.Content})
			}
		}
	}
	if len(systemParts) > 0 {
		apiMsgs = append([]chatMsg{{Role: "system", Content: strings.Join(systemParts, "\n")}}, apiMsgs...)
	}

	return func() tea.Msg {
		client := &http.Client{Timeout: 120 * time.Second}

		var reqBody []byte
		var req *http.Request
		var err error

		if agent.Provider == "anthropic" {
			type anthropicReq struct {
				Model     string    `json:"model"`
				Messages  []chatMsg `json:"messages"`
				MaxTokens int       `json:"max_tokens"`
			}
			reqBody, _ = json.Marshal(anthropicReq{
				Model:     agent.Model,
				Messages:  apiMsgs,
				MaxTokens: 4096,
			})
			req, err = http.NewRequestWithContext(ctx, "POST", base+"/v1/messages", strings.NewReader(string(reqBody)))
			if err != nil {
				return chatErrMsg{content: err.Error()}
			}
			req.Header.Set("x-api-key", key)
			req.Header.Set("anthropic-version", "2023-06-01")
			req.Header.Set("Content-Type", "application/json")
		} else if agent.Provider == "gemini" {
			type geminiPart struct {
				Text string `json:"text"`
			}
			type geminiContent struct {
				Parts []geminiPart `json:"parts"`
			}
			type geminiReq struct {
				Contents []geminiContent `json:"contents"`
			}
			var geminiMsgs []geminiContent
			for _, m := range apiMsgs {
				if strings.TrimSpace(m.Content) != "" {
					geminiMsgs = append(geminiMsgs, geminiContent{Parts: []geminiPart{{Text: m.Content}}})
				}
			}
			reqBody, _ = json.Marshal(geminiReq{Contents: geminiMsgs})
			url := fmt.Sprintf("%s/v1/models/%s:generateContent?key=%s", base, agent.Model, key)
			req, err = http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
			if err != nil {
				return chatErrMsg{content: err.Error()}
			}
			req.Header.Set("Content-Type", "application/json")
		} else {
			type openAIReq struct {
				Model     string    `json:"model"`
				Messages  []chatMsg `json:"messages"`
				MaxTokens int       `json:"max_tokens,omitempty"`
			}
			reqBody, _ = json.Marshal(openAIReq{Model: agent.Model, Messages: apiMsgs, MaxTokens: 4096})
			req, err = http.NewRequestWithContext(ctx, "POST", base+"/v1/chat/completions", strings.NewReader(string(reqBody)))
			if err != nil {
				return chatErrMsg{content: err.Error()}
			}
			req.Header.Set("Authorization", "Bearer "+key)
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err != nil {
			return chatErrMsg{content: err.Error()}
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return chatErrMsg{content: err.Error()}
		}

		if resp.StatusCode >= 400 {
			short := fmt.Sprintf("HTTP %d", resp.StatusCode)
			if resp.StatusCode == 401 {
				short += " Invalid API key"
			} else if resp.StatusCode == 403 {
				short += " Access denied"
			} else if resp.StatusCode == 429 {
				short += " Rate limited"
			} else if resp.StatusCode >= 500 {
				short += " Server error"
			} else {
				short += " Error"
			}
			return chatErrMsg{content: short}
		}

		if agent.Provider == "anthropic" {
			var anthropicResp struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			}
			if err := json.Unmarshal(body, &anthropicResp); err != nil {
				return chatErrMsg{content: "Failed to parse Anthropic response"}
			}
			if len(anthropicResp.Content) > 0 {
				return chatRespMsg{agentName: agent.Name, content: anthropicResp.Content[0].Text, provider: agent.Provider}
			}
			return chatErrMsg{content: "Empty Anthropic response"}
		}

		if agent.Provider == "gemini" {
			var geminiResp struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}
			if err := json.Unmarshal(body, &geminiResp); err != nil {
				return chatErrMsg{content: "Failed to parse Gemini response"}
			}
			if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
				return chatRespMsg{agentName: agent.Name, content: geminiResp.Candidates[0].Content.Parts[0].Text, provider: agent.Provider}
			}
			return chatErrMsg{content: "Empty Gemini response"}
		}

		var openAIResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &openAIResp); err != nil {
			return chatErrMsg{content: "Failed to parse response"}
		}
		if len(openAIResp.Choices) > 0 {
			return chatRespMsg{agentName: agent.Name, content: openAIResp.Choices[0].Message.Content, provider: agent.Provider}
		}
		return chatErrMsg{content: "Empty response"}
	}
}
