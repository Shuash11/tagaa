package main

import (
	"encoding/json"
	"os"
)

const configFile = "tagaa.config.json"

func saveConfig(m model) {
	data := savedConfig{APIKeys: m.apiKeys, Agents: m.agents}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(configFile, b, 0600)
}

func loadConfig() (map[string]string, []agentCfg) {
	b, err := os.ReadFile(configFile)
	if err != nil {
		return make(map[string]string), []agentCfg{}
	}
	var data savedConfig
	if err := json.Unmarshal(b, &data); err != nil {
		return make(map[string]string), []agentCfg{}
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
	// migration: remove empty-agent artifacts (old config with {} entries)
	clean := make([]agentCfg, 0, len(data.Agents))
	for _, a := range data.Agents {
		if a.Name != "" || a.Provider != "" || a.Model != "" {
			clean = append(clean, a)
		}
	}
	return data.APIKeys, clean
}
