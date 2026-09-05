package ai

import "context"

// MockProvider returns a hardcoded response for testing without an AI API.
type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error) {
	return &AnalysisResult{
		What:         "SQL query is built by concatenating user input, allowing injection.",
		Risk:         "Attacker can read, modify, or delete your entire database.",
		Fix:          "Replace string concatenation with parameterized queries.",
		ProposedCode: "cursor.execute(\"SELECT * FROM table WHERE id = %s\", [user_input])",
		CanAutoFix:   true,
	}, nil
}
