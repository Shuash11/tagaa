package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

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
			pid := providers[m.sidebarCur].id
			m.sidebarProv = pid
			m.agents[m.sidebarSel].Provider = pid
			m.agents[m.sidebarSel].Model = ""
			saveConfig(m)
			models := m.models[pid]
			if len(models) > 0 {
				m.sidebarStep = 2
				m.sidebarCur = 0
			} else {
				m.sidebarConfig = false
				m.messages = append(m.messages, Message{Kind: MsgSystem, Content: "No models loaded for " + pid + " — set API key in Ctrl+S Keys tab first"})
			}
			return m, nil
		} else if m.sidebarStep == 2 {
			models := m.models[m.sidebarProv]
			if m.sidebarCur < len(models) {
				m.agents[m.sidebarSel].Model = models[m.sidebarCur]
				saveConfig(m)
			}
			m.sidebarConfig = false
			return m, nil
		}
	}
	return m, nil
}
