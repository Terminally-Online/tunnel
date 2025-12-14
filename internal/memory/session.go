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

func EstimateTokens(text string) int {
	return len(text) / 4
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

func (s *Session) formatHistoryWithBudget(tokenBudget int) string {
	if len(s.History) == 0 || tokenBudget <= 0 {
		return ""
	}

	var parts []string
	totalTokens := 0

	for i := len(s.History) - 1; i >= 0; i-- {
		turn := s.History[i]
		var prefix string
		if turn.Role == "user" {
			prefix = "User: "
		} else {
			prefix = "Assistant: "
		}
		line := prefix + turn.Content + "\n"
		lineTokens := EstimateTokens(line)

		if totalTokens+lineTokens > tokenBudget {
			break
		}

		parts = append([]string{line}, parts...)
		totalTokens += lineTokens
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.TrimSpace(strings.Join(parts, ""))
}

func (s *Session) HistoryTokens() int {
	return EstimateTokens(s.FormatHistory())
}

func (s *Session) HistoryRatio(contextSize, maxTokens int, characterDescription string) float64 {
	if contextSize <= 0 {
		return 0
	}

	available := contextSize - maxTokens
	if available <= 0 {
		return 1.0
	}

	fixedTokens := s.fixedContextTokens(characterDescription)
	historyBudget := available - fixedTokens
	if historyBudget <= 0 {
		return 1.0
	}

	historyTokens := s.HistoryTokens()
	return float64(historyTokens) / float64(historyBudget)
}

func (s *Session) fixedContextTokens(characterDescription string) int {
	tokens := 0

	if characterDescription != "" {
		tokens += EstimateTokens("[Character]\n" + characterDescription + "\n\n")
	}
	if s.CharacterDeltas != "" {
		tokens += EstimateTokens("[Character Adaptations for This User]\n" + s.CharacterDeltas + "\n\n")
	}
	if s.Profile != "" {
		tokens += EstimateTokens("[User Profile]\n" + s.Profile + "\n\n")
	}
	if s.Summary != "" {
		tokens += EstimateTokens("[Summary of prior conversation]\n" + s.Summary + "\n\n")
	}
	tokens += EstimateTokens("[Recent exchange]\n\n")
	tokens += EstimateTokens("[Current message]\n")

	return tokens
}

func (s *Session) BuildPrompt(currentMessage string, characterDescription string, contextSize, maxTokens int) string {
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
		historyBudget := 0
		if contextSize > 0 {
			available := contextSize - maxTokens
			fixedTokens := s.fixedContextTokens(characterDescription)
			currentMsgTokens := EstimateTokens(currentMessage)
			historyBudget = available - fixedTokens - currentMsgTokens
		}

		var historyText string
		if historyBudget > 0 {
			historyText = s.formatHistoryWithBudget(historyBudget)
		} else if contextSize == 0 {
			historyText = s.FormatHistory()
		}

		if historyText != "" {
			sb.WriteString("[Recent exchange]\n")
			sb.WriteString(historyText)
			sb.WriteString("\n\n")
		}
	}

	sb.WriteString("[Current message]\n")
	sb.WriteString(currentMessage)

	return sb.String()
}
