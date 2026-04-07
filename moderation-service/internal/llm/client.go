package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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

type textResponse struct {
	Text string `json:"text"`
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

func (c *Client) Transcribe(ctx context.Context, text string) (string, error) {
	prompt := buildTranscribePrompt(text)
	raw, err := c.generateJSON(ctx, prompt)
	if err != nil {
		return "", err
	}

	parsed, err := parseTextJSON(raw)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(parsed.Text), nil
}

func (c *Client) Translate(ctx context.Context, text, targetLanguage string) (string, error) {
	prompt := buildTranslatePrompt(text, targetLanguage)
	raw, err := c.generateJSON(ctx, prompt)
	if err != nil {
		return "", err
	}

	parsed, err := parseTextJSON(raw)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(parsed.Text), nil
}

func (c *Client) TranscribeAudio(ctx context.Context, audioBase64, format, language string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("STT_PROVIDER")))
	if provider == "" {
		provider = "openai"
	}
	if provider != "openai" {
		return "", fmt.Errorf("unsupported STT_PROVIDER: %s", provider)
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is required for audio transcription")
	}

	baseURL := strings.TrimSuffix(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := strings.TrimSpace(os.Getenv("OPENAI_AUDIO_MODEL"))
	if model == "" {
		model = "gpt-4o-mini-transcribe"
	}

	audioBytes, err := decodeBase64Audio(audioBase64)
	if err != nil {
		return "", err
	}

	fileExt := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(format)), ".")
	if fileExt == "" {
		fileExt = "wav"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("model", model)
	if strings.TrimSpace(language) != "" {
		_ = writer.WriteField("language", strings.TrimSpace(language))
	}

	part, err := writer.CreateFormFile("file", "audio."+fileExt)
	if err != nil {
		return "", fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := part.Write(audioBytes); err != nil {
		return "", fmt.Errorf("write audio payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/audio/transcriptions", &body)
	if err != nil {
		return "", fmt.Errorf("create transcribe request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("call transcribe api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read transcribe response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("transcribe api status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("decode transcribe response: %w", err)
	}
	if strings.TrimSpace(out.Text) == "" {
		return "", fmt.Errorf("empty transcript returned")
	}

	return strings.TrimSpace(out.Text), nil
}

func (c *Client) generateJSON(ctx context.Context, prompt string) (string, error) {
	body := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}
	body.Options.Temperature = 0

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return "", fmt.Errorf("ollama status=%d body=%s", resp.StatusCode, string(body))
	}

	responseBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read ollama response: %w", err)
	}

	var raw generateResponse
	if err := json.Unmarshal(responseBytes, &raw); err != nil {
		return "", fmt.Errorf("decode ollama envelope: %w", err)
	}

	return raw.Response, nil
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

func buildTranscribePrompt(text string) string {
	return fmt.Sprintf(`You are a transcription normalizer.
Convert spoken or messy input into clean written text.

Rules:
- Output exactly one JSON object with this schema:
{
  "text": string
}
- Keep meaning unchanged.
- Fix punctuation, casing, and obvious speech disfluencies.
- No markdown, no comments, no explanation.

Input:
%s`, text)
}

func buildTranslatePrompt(text, targetLanguage string) string {
	return fmt.Sprintf(`You are a translation assistant.
Translate the input text into the target language.

Rules:
- Output exactly one JSON object with this schema:
{
  "text": string
}
- Keep meaning and tone as much as possible.
- No markdown, no comments, no explanation.

Target language:
%s

Input:
%s`, targetLanguage, text)
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

func parseTextJSON(raw string) (textResponse, error) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return textResponse{}, fmt.Errorf("llm did not return json")
	}
	jsonBlob := raw[start : end+1]

	var out textResponse
	if err := json.Unmarshal([]byte(jsonBlob), &out); err != nil {
		return textResponse{}, fmt.Errorf("invalid llm json: %w", err)
	}
	if strings.TrimSpace(out.Text) == "" {
		return textResponse{}, fmt.Errorf("llm returned empty text")
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

func decodeBase64Audio(input string) ([]byte, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, fmt.Errorf("audio_base64 is empty")
	}

	if strings.HasPrefix(raw, "data:") {
		comma := strings.Index(raw, ",")
		if comma == -1 {
			return nil, fmt.Errorf("invalid data URI audio payload")
		}
		raw = raw[comma+1:]
	}

	raw = strings.ReplaceAll(raw, "\n", "")
	raw = strings.ReplaceAll(raw, "\r", "")

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(raw)
		if err != nil {
			decoded, err = base64.URLEncoding.DecodeString(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid audio_base64 payload")
			}
		}
	}

	if len(decoded) == 0 {
		return nil, fmt.Errorf("decoded audio payload is empty")
	}

	// Simple sanity guard to reject obvious non-audio file paths accidentally sent.
	if strings.Contains(string(decoded), string(filepath.Separator)) && len(decoded) < 256 {
		return nil, fmt.Errorf("audio_base64 appears invalid")
	}

	return decoded, nil
}
