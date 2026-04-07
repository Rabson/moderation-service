package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"moderation-llm/moderation-service/internal/moderation"
)

type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

type generateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Format  string `json:"format,omitempty"`
	Options struct {
		Temperature float64 `json:"temperature"`
	} `json:"options"`
}

type generateResponse struct {
	Response string `json:"response"`
}

type llmResponse struct {
	Labels moderation.Labels `json:"labels"`
}

func NewClient(baseURL, model string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) Classify(ctx context.Context, text string) (moderation.Labels, error) {
	prompt := buildPrompt(text)

	body := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}
	body.Options.Temperature = 0

	payload, err := json.Marshal(body)
	if err != nil {
		return moderation.Labels{}, fmt.Errorf("marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return moderation.Labels{}, fmt.Errorf("create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return moderation.Labels{}, fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return moderation.Labels{}, fmt.Errorf("ollama status=%d body=%s", resp.StatusCode, string(body))
	}

	responseBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return moderation.Labels{}, fmt.Errorf("read ollama response: %w", err)
	}

	var raw generateResponse
	if err := json.Unmarshal(responseBytes, &raw); err != nil {
		return moderation.Labels{}, fmt.Errorf("decode ollama envelope: %w", err)
	}

	parsed, err := parseStrictJSON(raw.Response)
	if err != nil {
		return moderation.Labels{}, err
	}
	parsed.Labels = normalize(parsed.Labels)
	return parsed.Labels, nil
}

func buildPrompt(text string) string {
	return fmt.Sprintf(`You are a content moderation classifier.
Analyze the user text and return strict JSON only.

Rules:
- Output exactly one JSON object with this schema:
{
  "labels": {
    "hate": number,
    "violence": number,
    "sexual": number,
    "spam": number,
    "safe": number
  }
}
- Each score must be a float between 0 and 1.
- Multi-label allowed.
- Consider sarcasm, slang, obfuscation, and implied threats.
- "safe" should decrease as unsafe labels increase.
- No markdown, no comments, no explanation.

Text:
%s`, text)
}

func parseStrictJSON(raw string) (llmResponse, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return llmResponse{}, fmt.Errorf("llm did not return json")
	}
	jsonBlob := raw[start : end+1]

	var out llmResponse
	if err := json.Unmarshal([]byte(jsonBlob), &out); err != nil {
		return llmResponse{}, fmt.Errorf("invalid llm json: %w", err)
	}
	return out, nil
}

func normalize(l moderation.Labels) moderation.Labels {
	l.Hate = clamp01(l.Hate)
	l.Violence = clamp01(l.Violence)
	l.Sexual = clamp01(l.Sexual)
	l.Spam = clamp01(l.Spam)
	l.Safe = clamp01(l.Safe)
	return l
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
