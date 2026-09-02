package risk

import "github.com/lhilove/security-copilot/internal/database"

type Score struct {
	RiskScore int
	Critical  int
	High      int
	Medium    int
	Low       int
	Total     int
}

// Calculate computes a risk score from a finding summary.
// Score starts at 100 and is reduced by severity-weighted penalties.
// Critical findings are weighted most heavily since they represent
// immediate exploitable risk to the application.
func Calculate(s *database.FindingSummary) *Score {
	penalty := (s.Critical * 10) + (s.High * 5) + (s.Medium * 2) + (s.Low / 2)
	score := 100 - penalty
	if score < 0 {
		score = 0
	}

	return &Score{
		RiskScore: score,
		Critical:  s.Critical,
		High:      s.High,
		Medium:    s.Medium,
		Low:       s.Low,
		Total:     s.Total,
	}
}
