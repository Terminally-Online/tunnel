package memory

import (
	"strings"
	"time"
)

type Turn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Session struct {
	ID        string    `json:"id"`
	Summary   string    `json:"summary"`
	TurnCount int       `json:"turn_count"`
	History   []Turn    `json:"history"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Session) AddTurn(role, content string) {
	s.History = append(s.History, Turn{Role: role, Content: content})
	s.TurnCount++
	s.UpdatedAt = time.Now()
}

func (s *Session) ClearHistory() {
	s.History = nil
	s.TurnCount = 0
	s.UpdatedAt = time.Now()
}

func (s *Session) FormatHistory() string {
	if len(s.History) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, turn := range s.History {
		if turn.Role == "user" {
			sb.WriteString("User: ")
		} else {
			sb.WriteString("Assistant: ")
		}
		sb.WriteString(turn.Content)
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

func (s *Session) BuildPrompt(currentMessage string) string {
	var sb strings.Builder

	if s.Summary != "" {
		sb.WriteString("[Summary of prior conversation]\n")
		sb.WriteString(s.Summary)
		sb.WriteString("\n\n")
	}

	if len(s.History) > 0 {
		sb.WriteString("[Recent exchange]\n")
		sb.WriteString(s.FormatHistory())
		sb.WriteString("\n\n")
	}

	sb.WriteString("[Current message]\n")
	sb.WriteString(currentMessage)

	return sb.String()
}
