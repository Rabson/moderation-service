package moderation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"moderation-llm/moderation-service/internal/kafka"
	"moderation-llm/moderation-service/internal/storage"
)

type llmClient interface {
	Classify(ctx context.Context, text string) (Labels, error)
}

type cacheClient interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key string, value string) error
}

type Engine struct {
	rules      *RuleEngine
	llm        llmClient
	cache      cacheClient
	store      *storage.Postgres
	producer   *kafka.Producer
	logger     *slog.Logger
	llmTimeout time.Duration
}

type Event struct {
	RequestID string  `json:"request_id"`
	TextHash  string  `json:"text_hash"`
	RiskScore float64 `json:"risk_score"`
	Action    string  `json:"action"`
}

func NewEngine(
	rules *RuleEngine,
	llm llmClient,
	cache cacheClient,
	store *storage.Postgres,
	producer *kafka.Producer,
	logger *slog.Logger,
	llmTimeout time.Duration,
) *Engine {
	return &Engine{
		rules:      rules,
		llm:        llm,
		cache:      cache,
		store:      store,
		producer:   producer,
		logger:     logger,
		llmTimeout: llmTimeout,
	}
}

func (e *Engine) Moderate(ctx context.Context, requestID, text string) (Result, error) {
	preprocessed := Preprocess(text)
	if preprocessed == "" {
		return Result{}, errors.New("text is empty after preprocessing")
	}

	cacheKey := "moderation:" + hash(preprocessed)
	if cachedPayload, ok, err := e.cache.Get(ctx, cacheKey); err == nil && ok {
		var cached Result
		if err := json.Unmarshal([]byte(cachedPayload), &cached); err == nil {
			e.logOutcome(ctx, requestID, text, preprocessed, cached, "")
			return cached, nil
		}
	}

	ruleLabels := e.rules.Score(preprocessed)
	llmLabels, llmErr := e.classifyWithTimeout(ctx, preprocessed)

	labels := mergeLabels(ruleLabels, llmLabels)
	labels = normalizeLabels(labels)

	risk := calculateRisk(labels)
	result := Result{
		Labels:    labels,
		RiskScore: risk,
		Action:    decideAction(risk, labels),
	}

	cachedPayload, err := json.Marshal(result)
	if err != nil {
		e.logger.Warn("cache marshal failed", "error", err)
	} else if err := e.cache.Set(ctx, cacheKey, string(cachedPayload)); err != nil {
		e.logger.Warn("cache set failed", "error", err)
	}

	llmErrText := ""
	if llmErr != nil {
		llmErrText = llmErr.Error()
		e.logger.Warn("llm classification failed; used fallback", "error", llmErr)
	}
	e.logOutcome(ctx, requestID, text, preprocessed, result, llmErrText)

	if err := e.producer.Publish(ctx, requestID, Event{
		RequestID: requestID,
		TextHash:  hash(preprocessed),
		RiskScore: result.RiskScore,
		Action:    result.Action,
	}); err != nil {
		e.logger.Warn("kafka publish failed", "error", err)
	}

	return result, nil
}

func (e *Engine) classifyWithTimeout(ctx context.Context, text string) (Labels, error) {
	llmCtx, cancel := context.WithTimeout(ctx, e.llmTimeout)
	defer cancel()

	labels, err := e.llm.Classify(llmCtx, text)
	if err != nil {
		fallback := e.rules.Score(text)
		return fallback, err
	}
	return labels, nil
}

func (e *Engine) logOutcome(ctx context.Context, requestID, originalText, preprocessed string, result Result, llmErr string) {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	labelsJSON, err := json.Marshal(result.Labels)
	if err != nil {
		e.logger.Error("failed to marshal labels for logging", "error", err)
		return
	}

	err = e.store.InsertLog(dbCtx, storage.ModerationLog{
		RequestID:        requestID,
		OriginalText:     originalText,
		PreprocessedText: preprocessed,
		LabelsJSON:       string(labelsJSON),
		RiskScore:        result.RiskScore,
		Action:           result.Action,
		LLMError:         llmErr,
	})
	if err != nil {
		e.logger.Error("failed to store moderation log", "error", err)
	}
}

func calculateRisk(labels Labels) float64 {
	weighted := labels.Hate*0.4 + labels.Violence*0.3 + labels.Sexual*0.2 + labels.Spam*0.1
	dominant := max4(labels.Hate, labels.Violence, labels.Sexual, labels.Spam)

	// Keep weighted blending, but never let a strong single unsafe label look low.
	risk := max2(weighted, dominant)
	return round2(clamp01(risk))
}

// decideAction returns the moderation action based on the composite risk score
// AND the dominant (highest) individual label. This prevents a single high-severity
// label from being under-counted by the weighted formula.
// e.g. violence=0.9 now yields risk_score=0.9 and remains "block".
func decideAction(risk float64, labels Labels) string {
	dominant := max4(labels.Hate, labels.Violence, labels.Sexual, labels.Spam)

	// Dominant label overrides: checked before composite score.
	if dominant >= 0.85 {
		return "block"
	}
	if dominant >= 0.5 {
		return "review"
	}

	// Composite score thresholds.
	if risk < 0.3 {
		return "allow"
	}
	if risk <= 0.7 {
		return "review"
	}
	return "block"
}

func mergeLabels(a, b Labels) Labels {
	out := Labels{
		Hate:     max2(a.Hate, b.Hate),
		Violence: max2(a.Violence, b.Violence),
		Sexual:   max2(a.Sexual, b.Sexual),
		Spam:     max2(a.Spam, b.Spam),
		Safe:     max2(a.Safe, b.Safe),
	}

	maxUnsafe := max4(out.Hate, out.Violence, out.Sexual, out.Spam)
	if out.Safe > 1-maxUnsafe {
		out.Safe = 1 - maxUnsafe
	}
	return out
}

func normalizeLabels(l Labels) Labels {
	l.Hate = clamp01(l.Hate)
	l.Violence = clamp01(l.Violence)
	l.Sexual = clamp01(l.Sexual)
	l.Spam = clamp01(l.Spam)
	l.Safe = clamp01(l.Safe)
	return l
}

func hash(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}

func max2(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func max4(a, b, c, d float64) float64 {
	return max2(max2(a, b), max2(c, d))
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

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func (e *Engine) ModerateBatch(ctx context.Context, requestID string, texts []string) (BatchResult, error) {
	type indexedResult struct {
		idx int
		res Result
		err error
	}

	ch := make(chan indexedResult, len(texts))
	for i, text := range texts {
		i, text := i, text
		go func() {
			itemID := fmt.Sprintf("%s-%d", requestID, i)
			res, err := e.Moderate(ctx, itemID, text)
			ch <- indexedResult{idx: i, res: res, err: err}
		}()
	}

	results := make([]Result, len(texts))
	for range texts {
		r := <-ch
		if r.err != nil {
			return BatchResult{}, r.err
		}
		results[r.idx] = r.res
	}
	return BatchResult{Results: results}, nil
}
