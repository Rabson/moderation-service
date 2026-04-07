package moderation

type Labels struct {
	Hate     float64 `json:"hate"`
	Violence float64 `json:"violence"`
	Sexual   float64 `json:"sexual"`
	Spam     float64 `json:"spam"`
	Safe     float64 `json:"safe"`
}

type Result struct {
	Labels    Labels  `json:"labels"`
	RiskScore float64 `json:"risk_score"`
	Action    string  `json:"action"`
}

type BatchResult struct {
	Results []Result `json:"results"`
}

type Request struct {
	Text string `json:"text"`
}

type BatchRequest struct {
	Texts []string `json:"texts"`
}

type TranscribeRequest struct {
	Text string `json:"text"`
}

type TranscribeResult struct {
	Transcript string `json:"transcript"`
}

type TranslateRequest struct {
	Text           string `json:"text"`
	TargetLanguage string `json:"target_language"`
}

type TranslateResult struct {
	Text           string `json:"text"`
	TargetLanguage string `json:"target_language"`
}

type AudioTranscribeRequest struct {
	AudioBase64 string `json:"audio_base64"`
	Format      string `json:"format"`
	Language    string `json:"language"`
}

type AudioTranscribeResult struct {
	Transcript string `json:"transcript"`
	Language   string `json:"language,omitempty"`
}
