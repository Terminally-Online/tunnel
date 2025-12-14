package memory

import (
	"strings"
)

const DefaultSummaryPrompt = `Summarize this conversation concisely. Capture:
- Key facts mentioned (names, preferences, decisions)
- Behavioral patterns (how the user communicates, what they respond well to)
- Important context that should persist

Previous summary:
{previous_summary}

Recent exchange:
{history}

Write a new summary that incorporates both the previous context and recent exchange:`

func BuildSummarizationPrompt(session *Session, promptTemplate string) string {
	if promptTemplate == "" {
		promptTemplate = DefaultSummaryPrompt
	}

	previousSummary := session.Summary
	if previousSummary == "" {
		previousSummary = "(none)"
	}

	history := session.FormatHistory()
	if history == "" {
		history = "(none)"
	}

	result := promptTemplate
	result = strings.ReplaceAll(result, "{previous_summary}", previousSummary)
	result = strings.ReplaceAll(result, "{history}", history)

	return result
}
