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
	if n == 0 {
		return ""
	}
	for i := 0; i < n; i++ {
		idx := (m.msgAgentIdx + i) % n
		if idx < 0 {
			idx += n
		}
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
	if n == 0 {
		return func() tea.Msg {
			return chatErrMsg{content: "No ready agent: no agents configured"}
		}
	}
	for i := 0; i < n; i++ {
		idx := (m.msgAgentIdx + i) % n
		if idx < 0 {
			idx += n
		}
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

	// Build initial conversation messages from m.messages
	type openAIMsg struct {
		Role       string     `json:"role"`
		Content    string     `json:"content,omitempty"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
		ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	}
	type anthropicMsg struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
	}
	type geminiPart struct {
		Text             string                 `json:"text,omitempty"`
		FunctionCall     map[string]interface{} `json:"functionCall,omitempty"`
		FunctionResponse map[string]interface{} `json:"functionResponse,omitempty"`
	}
	type geminiContent struct {
		Role  string       `json:"role,omitempty"`
		Parts []geminiPart `json:"parts"`
	}

	var systemParts []string
	var openAIMsgs []openAIMsg
	var anthropicMsgs []anthropicMsg
	var geminiMsgs []geminiContent
	for _, msg := range m.messages {
		switch msg.Kind {
		case MsgSystem:
			if strings.TrimSpace(msg.Content) != "" {
				systemParts = append(systemParts, msg.Content)
			}
		case MsgUser:
			if strings.TrimSpace(msg.Content) != "" {
				openAIMsgs = append(openAIMsgs, openAIMsg{Role: "user", Content: msg.Content})
				anthropicMsgs = append(anthropicMsgs, anthropicMsg{Role: "user", Content: msg.Content})
				geminiMsgs = append(geminiMsgs, geminiContent{Role: "user", Parts: []geminiPart{{Text: msg.Content}}})
			}
		case MsgAgent:
			if strings.TrimSpace(msg.Content) != "" {
				openAIMsgs = append(openAIMsgs, openAIMsg{Role: "assistant", Content: msg.Content})
				anthropicMsgs = append(anthropicMsgs, anthropicMsg{Role: "assistant", Content: msg.Content})
				geminiMsgs = append(geminiMsgs, geminiContent{Role: "model", Parts: []geminiPart{{Text: msg.Content}}})
			}
		}
	}
	if len(systemParts) > 0 {
		sys := strings.Join(systemParts, "\n")
		openAIMsgs = append([]openAIMsg{{Role: "system", Content: sys}}, openAIMsgs...)
		anthropicMsgs = append([]anthropicMsg{{Role: "user", Content: sys}}, anthropicMsgs...)
	}

	return func() tea.Msg {
		client := &http.Client{Timeout: 120 * time.Second}
		toolsCfg := formatToolsForProvider(agent.Provider)

		var reqBody []byte
		var req *http.Request
		var err error

		switch agent.Provider {
		case "anthropic":
			type anthropicReq struct {
				Model     string                   `json:"model"`
				Messages  []anthropicMsg           `json:"messages"`
				MaxTokens int                      `json:"max_tokens"`
				Tools     []map[string]interface{} `json:"tools,omitempty"`
			}
			reqBody, _ = json.Marshal(anthropicReq{Model: agent.Model, Messages: anthropicMsgs, MaxTokens: 4096, Tools: toolsCfg.([]map[string]interface{})})
			req, err = http.NewRequestWithContext(ctx, "POST", base+"/v1/messages", strings.NewReader(string(reqBody)))
			if err != nil {
				return chatErrMsg{content: err.Error()}
			}
			req.Header.Set("x-api-key", key)
			req.Header.Set("anthropic-version", "2023-06-01")
			req.Header.Set("Content-Type", "application/json")

		case "gemini":
			type geminiReq struct {
				Contents []geminiContent          `json:"contents"`
				Tools    []map[string]interface{} `json:"tools,omitempty"`
			}
			reqBody, _ = json.Marshal(geminiReq{Contents: geminiMsgs, Tools: toolsCfg.([]map[string]interface{})})
			url := fmt.Sprintf("%s/v1/models/%s:generateContent?key=%s", base, agent.Model, key)
			req, err = http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
			if err != nil {
				return chatErrMsg{content: err.Error()}
			}
			req.Header.Set("Content-Type", "application/json")

		default:
			type openAIReq struct {
				Model     string                   `json:"model"`
				Messages  []openAIMsg              `json:"messages"`
				MaxTokens int                      `json:"max_tokens,omitempty"`
				Tools     []map[string]interface{} `json:"tools,omitempty"`
			}
			reqBody, _ = json.Marshal(openAIReq{Model: agent.Model, Messages: openAIMsgs, MaxTokens: 4096, Tools: toolsCfg.([]map[string]interface{})})
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
			switch resp.StatusCode {
			case 401:
				short += " Invalid API key"
			case 403:
				short += " Access denied"
			case 429:
				short += " Rate limited"
			default:
				if resp.StatusCode >= 500 {
					short += " Server error"
				} else {
					short += " Error"
				}
			}
			return chatErrMsg{content: short}
		}

		// Check for tool calls using the unified parser
		calls := parseToolCalls(agent.Provider, body)
		if len(calls) == 0 {
			content := extractFinalText(agent.Provider, body)
			if content == "" {
				return chatErrMsg{content: "Empty response"}
			}
			return chatRespMsg{agentName: agent.Name, content: content, provider: agent.Provider}
		}

		// Tool calls present — execute and follow up
		var toolSummary string
		for _, tc := range calls {
			result := executeToolCall(tc.Function.Name, tc.Function.Arguments)
			switch agent.Provider {
			case "anthropic":
				var argsRaw interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &argsRaw)
				anthropicMsgs = append(anthropicMsgs, anthropicMsg{
					Role: "assistant",
					Content: []interface{}{
						map[string]interface{}{"type": "tool_use", "id": tc.ID, "name": tc.Function.Name, "input": argsRaw},
					},
				})
				anthropicMsgs = append(anthropicMsgs, anthropicMsg{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{"type": "tool_result", "tool_use_id": tc.ID, "content": result},
					},
				})
			case "gemini":
				var argsRaw map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &argsRaw)
				geminiMsgs = append(geminiMsgs, geminiContent{
					Role: "model",
					Parts: []geminiPart{{FunctionCall: map[string]interface{}{"name": tc.Function.Name, "args": argsRaw}}},
				})
				geminiMsgs = append(geminiMsgs, geminiContent{
					Role: "function",
					Parts: []geminiPart{{FunctionResponse: map[string]interface{}{"name": tc.Function.Name, "response": map[string]interface{}{"result": result}}}},
				})
			default:
				openAIMsgs = append(openAIMsgs, openAIMsg{Role: "assistant", ToolCalls: []toolCall{tc}})
				openAIMsgs = append(openAIMsgs, openAIMsg{Role: "tool", ToolCallID: tc.ID, Content: result})
			}
			if strings.HasPrefix(result, "Successfully") {
				toolSummary = result
			}
		}

		// Follow-up request
		switch agent.Provider {
		case "anthropic":
			type anthropicReq struct {
				Model     string         `json:"model"`
				Messages  []anthropicMsg `json:"messages"`
				MaxTokens int            `json:"max_tokens"`
			}
			reqBody, _ = json.Marshal(anthropicReq{Model: agent.Model, Messages: anthropicMsgs, MaxTokens: 4096})
			req, err = http.NewRequestWithContext(ctx, "POST", base+"/v1/messages", strings.NewReader(string(reqBody)))
			req.Header.Set("x-api-key", key)
			req.Header.Set("anthropic-version", "2023-06-01")
			req.Header.Set("Content-Type", "application/json")
		case "gemini":
			type geminiReq struct {
				Contents []geminiContent `json:"contents"`
			}
			reqBody, _ = json.Marshal(geminiReq{Contents: geminiMsgs})
			url := fmt.Sprintf("%s/v1/models/%s:generateContent?key=%s", base, agent.Model, key)
			req, err = http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
			req.Header.Set("Content-Type", "application/json")
		default:
			type openAIReq struct {
				Model     string      `json:"model"`
				Messages  []openAIMsg `json:"messages"`
				MaxTokens int         `json:"max_tokens,omitempty"`
			}
			reqBody, _ = json.Marshal(openAIReq{Model: agent.Model, Messages: openAIMsgs, MaxTokens: 4096})
			req, err = http.NewRequestWithContext(ctx, "POST", base+"/v1/chat/completions", strings.NewReader(string(reqBody)))
			req.Header.Set("Authorization", "Bearer "+key)
			req.Header.Set("Content-Type", "application/json")
		}
		if err != nil {
			if toolSummary != "" {
				return chatRespMsg{agentName: agent.Name, content: toolSummary + "\n\n(Follow-up error)", provider: agent.Provider, toolResult: toolSummary}
			}
			return chatErrMsg{content: "Follow-up request failed: " + err.Error()}
		}

		resp2, err := client.Do(req)
		if err != nil {
			if toolSummary != "" {
				return chatRespMsg{agentName: agent.Name, content: toolSummary + "\n\n(Follow-up failed)", provider: agent.Provider, toolResult: toolSummary}
			}
			return chatErrMsg{content: "Follow-up call failed: " + err.Error()}
		}
		defer resp2.Body.Close()
		body2, err := io.ReadAll(resp2.Body)
		if err != nil {
			if toolSummary != "" {
				return chatRespMsg{agentName: agent.Name, content: toolSummary + "\n\n(Read error)", provider: agent.Provider, toolResult: toolSummary}
			}
			return chatErrMsg{content: "Failed to read follow-up response"}
		}
		content := extractFinalText(agent.Provider, body2)
		if content == "" && toolSummary != "" {
			return chatRespMsg{agentName: agent.Name, content: toolSummary, provider: agent.Provider, toolResult: toolSummary}
		}
		if content == "" {
			return chatErrMsg{content: "Empty follow-up response"}
		}
		return chatRespMsg{agentName: agent.Name, content: content, provider: agent.Provider, toolResult: toolSummary}
	}
}
