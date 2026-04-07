package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"moderation-llm/moderation-service/internal/moderation"
)

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) moderate(w http.ResponseWriter, r *http.Request) {
	var req moderation.Request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}

	result, err := s.engine.Moderate(r.Context(), middleware.GetReqID(r.Context()), req.Text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) moderateBatch(w http.ResponseWriter, r *http.Request) {
	var req moderation.BatchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Texts) == 0 {
		writeError(w, http.StatusBadRequest, "texts is required")
		return
	}
	if len(req.Texts) > 100 {
		writeError(w, http.StatusBadRequest, "batch size cannot exceed 100")
		return
	}

	for _, text := range req.Texts {
		if strings.TrimSpace(text) == "" {
			writeError(w, http.StatusBadRequest, "texts cannot contain empty values")
			return
		}
	}

	result, err := s.engine.ModerateBatch(r.Context(), middleware.GetReqID(r.Context()), req.Texts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) transcribe(w http.ResponseWriter, r *http.Request) {
	var req moderation.TranscribeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}

	result, err := s.engine.Transcribe(r.Context(), req.Text)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) transcribeAudio(w http.ResponseWriter, r *http.Request) {
	var req moderation.AudioTranscribeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.AudioBase64 = strings.TrimSpace(req.AudioBase64)
	req.Format = strings.TrimSpace(req.Format)
	req.Language = strings.TrimSpace(req.Language)

	if req.AudioBase64 == "" {
		writeError(w, http.StatusBadRequest, "audio_base64 is required")
		return
	}

	result, err := s.engine.TranscribeAudio(r.Context(), req.AudioBase64, req.Format, req.Language)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) translate(w http.ResponseWriter, r *http.Request) {
	var req moderation.TranslateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	req.TargetLanguage = strings.TrimSpace(req.TargetLanguage)
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	if req.TargetLanguage == "" {
		writeError(w, http.StatusBadRequest, "target_language is required")
		return
	}

	result, err := s.engine.Translate(r.Context(), req.Text, req.TargetLanguage)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
