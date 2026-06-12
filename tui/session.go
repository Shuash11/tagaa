package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type SavedMsg struct {
	Kind      int    `json:"kind"`
	AgentName string `json:"agent_name,omitempty"`
	Color     string `json:"color,omitempty"`
	Content   string `json:"content"`
}

type Session struct {
	ID        int         `json:"id"`
	Timestamp string      `json:"timestamp"`
	Messages  []SavedMsg  `json:"messages"`
}

type SessionsFile struct {
	LastID   int       `json:"last_id"`
	Sessions []Session `json:"sessions"`
}

const sessionFile = "tagaa.sessions.json"

func saveSessions(messages []Message) {
	var sf SessionsFile
	b, err := os.ReadFile(sessionFile)
	if err == nil {
		json.Unmarshal(b, &sf)
	}

	saved := make([]SavedMsg, 0, len(messages))
	for _, msg := range messages {
		saved = append(saved, SavedMsg{
			Kind:      int(msg.Kind),
			AgentName: msg.AgentName,
			Color:     string(msg.Color),
			Content:   msg.Content,
		})
	}

	sf.LastID++
	sf.Sessions = append(sf.Sessions, Session{
		ID:        sf.LastID,
		Timestamp: time.Now().Format("2006-01-02 15:04"),
		Messages:  saved,
	})

	if len(sf.Sessions) > 10 {
		sf.Sessions = sf.Sessions[len(sf.Sessions)-10:]
	}

	b, _ = json.MarshalIndent(sf, "", "  ")
	os.WriteFile(sessionFile, b, 0600)
}

func loadLatestSession() (int, string, []Message) {
	b, err := os.ReadFile(sessionFile)
	if err != nil {
		return 0, "", nil
	}
	var sf SessionsFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return 0, "", nil
	}
	if len(sf.Sessions) == 0 {
		return sf.LastID, "", nil
	}
	last := sf.Sessions[len(sf.Sessions)-1]
	msgs := make([]Message, len(last.Messages))
	for i, sm := range last.Messages {
		msgs[i] = Message{
			Kind:      MsgKind(sm.Kind),
			AgentName: sm.AgentName,
			Color:     lipgloss.Color(sm.Color),
			Content:   sm.Content,
		}
	}
	return last.ID, last.Timestamp, msgs
}
