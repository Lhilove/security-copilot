package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "deepseek-coder:6.7b"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 3 * time.Minute},
	}
}

type ollamaRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Stream  bool          `json:"stream"`
	Options ollamaOptions `json:"options"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func (p *OllamaProvider) Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error) {
	if err := validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid analysis request: %w", err)
	}

	// Build a conversational prompt that deepseek-coder responds to correctly
	fullPrompt := fmt.Sprintf(`%s

	### Security Finding to Analyze:
	Source: %s
	Severity: %s
	Title: %s
	Package: %s
	CVE: %s
	Description: %s

	### Your JSON Response:`,
		systemPrompt(),
		req.Source,
		req.Severity,
		req.Title,
		req.PackageName,
		req.CVEID,
		req.Description,
	)

	body := ollamaRequest{
		Model:  p.model,
		Prompt: fullPrompt,
		Stream: false,
		Options: ollamaOptions{
			Temperature: 0.1,
			NumPredict:  512,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/api/generate", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create ollama request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(b))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	if ollamaResp.Response == "" {
		return nil, fmt.Errorf("empty response from ollama")
	}

	log.Printf("ollama raw response: %s", ollamaResp.Response)
	return parseResponse(ollamaResp.Response), nil
}
