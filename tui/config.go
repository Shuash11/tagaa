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
	return data.APIKeys, data.Agents
}
