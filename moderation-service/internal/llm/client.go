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
	provider string
	baseURL  string
	model    string
	apiKey   string
	http     *http.Client
}

type providerConfig struct {
	Provider string
	BaseURL  string
	Model    string
	APIKey   string
}

type taskType string

const (
	taskModerate        taskType = "moderate"
	taskTranscribe      taskType = "transcribe"
	taskTranscribeAudio taskType = "transcribe_audio"
	taskTranslate       taskType = "translate"
)

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

type openAIChatRequest struct {
	Model          string              `json:"model"`
	Messages       []openAIMessage     `json:"messages"`
	Temperature    float64             `json:"temperature"`
	ResponseFormat *openAIResponseType `json:"response_format,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseType struct {
	Type string `json:"type"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type googleGenerateContentRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	GenerationConfig struct {
		Temperature      float64 `json:"temperature"`
		ResponseMimeType string  `json:"responseMimeType,omitempty"`
	} `json:"generationConfig"`
}

type googleGenerateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type llmResponse struct {
	Labels moderation.Labels `json:"labels"`
}

type textResponse struct {
	Text string `json:"text"`
}

func NewClient(timeout time.Duration) (*Client, error) {
	c := &Client{
		http: &http.Client{Timeout: timeout},
	}

	for _, task := range []taskType{taskModerate, taskTranscribe, taskTranscribeAudio, taskTranslate} {
		if _, err := c.providerConfigForTask(task); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Client) Classify(ctx context.Context, text string) (moderation.Labels, error) {
	prompt := buildPrompt(text)

	raw, err := c.generateJSONForTask(ctx, prompt, taskModerate)
	if err != nil {
		return moderation.Labels{}, err
	}

	parsed, err := parseStrictJSON(raw)
	if err != nil {
		return moderation.Labels{}, err
	}
	parsed.Labels = normalize(parsed.Labels)
	return parsed.Labels, nil
}

func (c *Client) Transcribe(ctx context.Context, text string) (string, error) {
	prompt := buildTranscribePrompt(text)
	raw, err := c.generateJSONForTask(ctx, prompt, taskTranscribe)
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
	raw, err := c.generateJSONForTask(ctx, prompt, taskTranslate)
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
	cfg, err := c.providerConfigForTask(taskTranscribeAudio)
	if err != nil {
		return "", err
	}
	switch cfg.Provider {
	case "openai":
		return c.transcribeAudioWithOpenAI(ctx, audioBase64, format, language, cfg)
	default:
		return "", fmt.Errorf("unsupported audio transcription provider: %s", cfg.Provider)
	}
}

func (c *Client) transcribeAudioWithOpenAI(ctx context.Context, audioBase64, format, language string, cfg providerConfig) (string, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return "", fmt.Errorf("LLM_TRANSCRIBE_AUDIO_API_KEY or OPENAI_API_KEY is required for audio transcription")
	}

	baseURL := strings.TrimSuffix(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := strings.TrimSpace(cfg.Model)
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
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
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

func (c *Client) generateJSONForTask(ctx context.Context, prompt string, task taskType) (string, error) {
	cfg, err := c.providerConfigForTask(task)
	if err != nil {
		return "", err
	}
	taskClient := *c
	taskClient.provider = cfg.Provider
	taskClient.baseURL = cfg.BaseURL
	taskClient.model = cfg.Model
	taskClient.apiKey = cfg.APIKey
	return taskClient.generateJSON(ctx, prompt)
}

func (c *Client) providerConfigForTask(task taskType) (providerConfig, error) {
	prefix := "LLM_" + strings.ToUpper(string(task)) + "_"

	defaultProvider, defaultBaseURL, defaultModel := defaultTaskSettings(task)
	defaultAPIKey := ""

	provider := strings.ToLower(strings.TrimSpace(getEnvOrFallback(prefix+"PROVIDER", defaultProvider)))
	if provider == "" {
		provider = "ollama"
	}

	if _, ok := os.LookupEnv(prefix + "PROVIDER"); ok {
		if err := validateProvider(prefix+"PROVIDER", provider); err != nil {
			return providerConfig{}, err
		}
	}

	baseURL := strings.TrimSuffix(strings.TrimSpace(getEnvOrFallback(prefix+"BASE_URL", defaultBaseURL)), "/")
	model := strings.TrimSpace(getEnvOrFallback(prefix+"MODEL", defaultModel))
	apiKey := strings.TrimSpace(getEnvOrFallback(prefix+"API_KEY", defaultAPIKey))

	switch provider {
	case "openai":
		if baseURL == "" {
			baseURL = strings.TrimSuffix(strings.TrimSpace(getEnvOrFallback("OPENAI_BASE_URL", "https://api.openai.com/v1")), "/")
		}
		if apiKey == "" {
			apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		}
	case "google":
		if baseURL == "" {
			baseURL = strings.TrimSuffix(strings.TrimSpace(getEnvOrFallback("GOOGLE_GENAI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta")), "/")
		}
		if apiKey == "" {
			apiKey = strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
		}
	case "ollama":
		if baseURL == "" {
			baseURL = "http://ollama:11434"
		}
	}

	return providerConfig{
		Provider: provider,
		BaseURL:  baseURL,
		Model:    model,
		APIKey:   apiKey,
	}, nil
}

func defaultTaskSettings(task taskType) (provider, baseURL, model string) {
	switch task {
	case taskTranscribeAudio:
		return "openai", "https://api.openai.com/v1", "gpt-4o-mini-transcribe"
	case taskModerate, taskTranscribe, taskTranslate:
		return "ollama", "http://ollama:11434", "gemma:2b"
	default:
		return "ollama", "http://ollama:11434", "gemma:2b"
	}
}

func validateProvider(envKey, provider string) error {
	switch provider {
	case "ollama", "openai", "google":
		return nil
	default:
		return fmt.Errorf("invalid %s=%q; allowed values: ollama, openai, google", envKey, provider)
	}
}

func getEnvOrFallback(key, fallback string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return v
}

func (c *Client) generateJSON(ctx context.Context, prompt string) (string, error) {
	switch c.provider {
	case "ollama":
		return c.generateJSONWithOllama(ctx, prompt)
	case "openai":
		return c.generateJSONWithOpenAI(ctx, prompt)
	case "google":
		return c.generateJSONWithGoogle(ctx, prompt)
	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", c.provider)
	}
}

func (c *Client) generateJSONWithOllama(ctx context.Context, prompt string) (string, error) {
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

func (c *Client) generateJSONWithOpenAI(ctx context.Context, prompt string) (string, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", fmt.Errorf("LLM_API_KEY is required for openai provider")
	}

	body := openAIChatRequest{
		Model:       c.model,
		Messages:    []openAIMessage{{Role: "user", Content: prompt}},
		Temperature: 0,
		ResponseFormat: &openAIResponseType{
			Type: "json_object",
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("call openai: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read openai response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var out openAIChatResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("decode openai response: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	content := extractOpenAIContent(out.Choices[0].Message.Content)
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("openai returned empty content")
	}

	return content, nil
}

func (c *Client) generateJSONWithGoogle(ctx context.Context, prompt string) (string, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", fmt.Errorf("LLM_API_KEY is required for google provider")
	}

	body := googleGenerateContentRequest{}
	body.Contents = []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{
		{
			Parts: []struct {
				Text string `json:"text"`
			}{
				{Text: prompt},
			},
		},
	}
	body.GenerationConfig.Temperature = 0
	body.GenerationConfig.ResponseMimeType = "application/json"

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal google request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create google request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("call google: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read google response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("google status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var out googleGenerateContentResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("decode google response: %w", err)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("google returned no candidates")
	}

	content := strings.TrimSpace(out.Candidates[0].Content.Parts[0].Text)
	if content == "" {
		return "", fmt.Errorf("google returned empty content")
	}

	return content, nil
}

func extractOpenAIContent(content any) string {
	switch v := content.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := obj["text"].(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
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
