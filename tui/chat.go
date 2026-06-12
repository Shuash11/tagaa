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
		client := &http.Client{Timeout: 20 * time.Second}
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
			if len(short) > 80 {
				short = short[:80] + "…"
			}
			return fetchModelsMsg{provider: id, err: errors.New(short)}
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fetchModelsMsg{provider: id, err: err}
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
			bodyStr := string(body)
			if len(bodyStr) > 60 {
				bodyStr = bodyStr[:60]
			}
			return fetchModelsMsg{provider: id, err: fmt.Errorf("%s: %s", short, bodyStr)}
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
		bodyStr := string(body)
			if len(bodyStr) > 80 {
				bodyStr = bodyStr[:80] + "..."
			}
			return fetchModelsMsg{provider: id, err: fmt.Errorf("unrecognized: %s", bodyStr)}
	}
}

func (m model) orchestrator() (agentCfg, bool) {
	for _, a := range m.agents {
		if a.IsOrchestrator && a.Enabled && a.Provider != "" && a.Model != "" && m.apiKeys[a.Provider] != "" {
			return a, true
		}
	}
	return agentCfg{}, false
}

func (m model) readyAgents() []agentCfg {
	var agents []agentCfg
	for _, a := range m.agents {
		if a.Enabled && a.Provider != "" && a.Model != "" && m.apiKeys[a.Provider] != "" {
			agents = append(agents, a)
		}
	}
	return agents
}

func (m model) workers() []agentCfg {
	var workers []agentCfg
	for _, a := range m.readyAgents() {
		if !a.IsOrchestrator {
			workers = append(workers, a)
		}
	}
	return workers
}

func sendGroupThinkCmd(m model, ctx context.Context) tea.Cmd {
	agents := m.readyAgents()
	if len(agents) == 0 {
		msg := "No ready agent:"
		if len(m.agents) == 0 {
			msg += " no agents configured"
		}
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
			return chatErrMsg{content: msg + "\n  → Press Ctrl+S to add an API key and configure an agent"}
		}
	}

	var task string
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Kind == MsgUser && m.messages[i].Content != "" {
			task = m.messages[i].Content
			break
		}
	}

	if !isTaskRequest(task) && len(agents) > 0 {
		a := agents[0]
		if orch, ok := m.orchestrator(); ok {
			a = orch
		}
		return func() tea.Msg {
			text, err := queryAgentSimple(ctx, a, m.apiKeys, "You are a helpful AI assistant. Be conversational and concise.", task)
			if err != nil {
				return chatErrMsg{agentName: a.Name, content: err.Error()}
			}
			return chatRespMsg{agentName: a.Name, content: text, provider: a.Provider}
		}
	}

	orch, hasOrch := m.orchestrator()
	workers := m.workers()

	ch := m.pipelineCh
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- pipelineBatchMsg{
					phase:    "error",
					messages: []Message{{Kind: MsgError, Content: fmt.Sprintf("Pipeline panic: %v", r)}},
					done:     true,
				}
			}
		}()
		p := NewPipeline(agents, m.apiKeys)
		if hasOrch {
			p.orchestrator = orch
		}
		p.workers = workers
		historyCopy := make([]Message, len(m.messages))
		copy(historyCopy, m.messages)
		p.history = historyCopy
		p.Run(ctx, task, ch)
	}()
	return pipelinePollCmd(ch)
}

// sendOrchMessage sends a direct message to the orchestrator and returns a tea.Cmd
func sendOrchMessage(m model, ctx context.Context, text string) tea.Cmd {
	if orch, ok := m.orchestrator(); ok {
		return func() tea.Msg {
			resp, err := queryAgentSimple(ctx, orch, m.apiKeys, "You are an orchestrator.", text)
			if err != nil {
				return chatErrMsg{agentName: orch.Name, content: err.Error()}
			}
			return chatRespMsg{agentName: orch.Name, content: resp, provider: orch.Provider}
		}
	}
	return func() tea.Msg { return chatErrMsg{content: "No orchestrator configured"} }
}

func pipelinePollCmd(ch chan pipelineBatchMsg) tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		select {
		case msg, ok := <-ch:
			if !ok {
				return pipelineBatchMsg{done: true}
			}
			return msg
		default:
			return pipelineTickMsg{}
		}
	})
}