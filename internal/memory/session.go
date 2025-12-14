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
	ID                 string    `json:"id"`
	Summary            string    `json:"summary"`
	Profile            string    `json:"profile"`
	CharacterDeltas    string    `json:"character_deltas"`
	TurnCount          int       `json:"turn_count"`
	TotalTurnCount     int       `json:"total_turn_count"`
	History            []Turn    `json:"history"`
	UpdatedAt          time.Time `json:"updated_at"`
	CharacterUpdatedAt time.Time `json:"character_updated_at"`
}

func (s *Session) AddTurn(role, content string) {
	s.History = append(s.History, Turn{Role: role, Content: content})
	s.TurnCount++
	s.TotalTurnCount++
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

func (s *Session) BuildPrompt(currentMessage string, characterDescription string) string {
	var sb strings.Builder

	if characterDescription != "" {
		sb.WriteString("[Character]\n")
		sb.WriteString(characterDescription)
		sb.WriteString("\n\n")
	}

	if s.CharacterDeltas != "" {
		sb.WriteString("[Character Adaptations for This User]\n")
		sb.WriteString(s.CharacterDeltas)
		sb.WriteString("\n\n")
	}

	if s.Profile != "" {
		sb.WriteString("[User Profile]\n")
		sb.WriteString(s.Profile)
		sb.WriteString("\n\n")
	}

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
