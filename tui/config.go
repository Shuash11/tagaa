package main

import (
	"encoding/json"
	"os"
	"strings"
)

const configFile = "tagaa.config.json"

var tokenCosts = map[string]struct{ input, output float64 }{
	"claude-sonnet-4-20250514":   {3.00, 15.00},
	"claude-3-5-sonnet-20241022": {3.00, 15.00},
	"claude-3-haiku-20240307":    {0.25, 1.25},
	"claude-opus-4-20250514":     {15.00, 75.00},
	"gpt-4o":                     {2.50, 10.00},
	"gpt-4o-mini":                {0.15, 0.60},
	"gpt-4-turbo":                {10.00, 30.00},
	"gemini-2.0-flash":           {0.10, 0.40},
	"gemini-1.5-pro":             {1.25, 5.00},
	"mistral-large-latest":       {2.00, 6.00},
	"deepseek-chat":              {0.27, 1.10},
	"grok-2":                     {2.00, 10.00},
}

func saveConfig(m model) {
	apiKeys := make(map[string]string, len(m.apiKeys))
	for k, v := range m.apiKeys {
		if k == "gemini" {
			apiKeys["google"] = v
		} else {
			apiKeys[k] = v
		}
	}
	agents := make([]agentCfg, len(m.agents))
	for i, a := range m.agents {
		agents[i] = a
		if a.Provider == "gemini" {
			agents[i].Provider = "google"
		}
	}
	data := savedConfig{APIKeys: apiKeys, Agents: agents, ShowTokenEstimate: m.showTokenEstimate}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(configFile, b, 0600)
}

func loadConfig() (map[string]string, []agentCfg, bool) {
	b, err := os.ReadFile(configFile)
	if err != nil {
		return make(map[string]string), []agentCfg{}, false
	}
	var data savedConfig
	if err := json.Unmarshal(b, &data); err != nil {
		return make(map[string]string), []agentCfg{}, false
	}
	if data.APIKeys == nil {
		data.APIKeys = make(map[string]string)
	}
	// migration: old config format had unexported fields,
	// so Enabled defaulted to false — auto-enable any agent with a name
	for i := range data.Agents {
		if data.Agents[i].Name != "" && !data.Agents[i].Enabled {
			data.Agents[i].Enabled = true
		}
	}
	// migration: TypeScript config uses "google" but TUI uses "gemini"
	if key, ok := data.APIKeys["google"]; ok {
		delete(data.APIKeys, "google")
		data.APIKeys["gemini"] = key
	}
	for i := range data.Agents {
		if data.Agents[i].Provider == "google" {
			data.Agents[i].Provider = "gemini"
		}
	}

	// resolve ${ENV_VAR} placeholders in API keys
	for k, v := range data.APIKeys {
		if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
			envName := v[2 : len(v)-1]
			resolved := os.Getenv(envName)
			if resolved != "" {
				data.APIKeys[k] = resolved
			} else {
				data.APIKeys[k] = ""
			}
		}
	}

	// migration: remove empty-agent artifacts (old config with {} entries)
	clean := make([]agentCfg, 0, len(data.Agents))
	for _, a := range data.Agents {
		if a.Name != "" || a.Provider != "" || a.Model != "" {
			clean = append(clean, a)
		}
	}
	return data.APIKeys, clean, data.ShowTokenEstimate
}
