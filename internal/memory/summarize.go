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

const DefaultProfilePrompt = `You are extracting stable user preferences from a conversation.

Task:
- Identify persistent preferences or interaction patterns.
- Ignore temporary moods or one-off statements.
- Write in short, neutral bullet points.
- Do NOT speculate.
- Max 12 bullets.

Conversation:
{history}

Existing User Profile (if any):
{existing_profile}

Output only the updated profile.`

const DefaultCharacterEvolutionPrompt = `You are refining how a fictional character behaves with a specific user.

Character Base Description:
{character_description}

Conversation Summary:
{summary}

User Profile:
{profile}

Task:
- Describe how the character's behavior with THIS user differs from default.
- Keep it factual and concise.
- Use 1–3 bullet points.
- Do NOT restate the base character.
- Do NOT roleplay.

Output only the bullet points.`

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

func BuildProfileExtractionPrompt(session *Session, promptTemplate string) string {
	if promptTemplate == "" {
		promptTemplate = DefaultProfilePrompt
	}

	existingProfile := session.Profile
	if existingProfile == "" {
		existingProfile = "(none)"
	}

	history := session.FormatHistory()
	if history == "" {
		history = "(none)"
	}

	result := promptTemplate
	result = strings.ReplaceAll(result, "{existing_profile}", existingProfile)
	result = strings.ReplaceAll(result, "{history}", history)

	return result
}

func BuildCharacterEvolutionPrompt(session *Session, characterDescription string) string {
	summary := session.Summary
	if summary == "" {
		summary = "(none)"
	}

	profile := session.Profile
	if profile == "" {
		profile = "(none)"
	}

	result := DefaultCharacterEvolutionPrompt
	result = strings.ReplaceAll(result, "{character_description}", characterDescription)
	result = strings.ReplaceAll(result, "{summary}", summary)
	result = strings.ReplaceAll(result, "{profile}", profile)

	return result
}
