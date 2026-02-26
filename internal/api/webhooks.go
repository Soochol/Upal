package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if s.triggerRepo == nil {
		http.Error(w, "triggers not available", http.StatusServiceUnavailable)
		return
	}

	trigger, err := s.triggerRepo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "trigger not found", http.StatusNotFound)
		return
	}
	if !trigger.Enabled {
		http.Error(w, "trigger is disabled", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	if trigger.Config.Secret != "" {
		signature := r.Header.Get("X-Webhook-Signature")
		if !verifyHMAC(body, trigger.Config.Secret, signature) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	var payload map[string]any
	if len(body) > 0 {
		json.Unmarshal(body, &payload)
	}

	inputs := mapInputs(payload, trigger.Config.InputMapping)

	if trigger.PipelineID != "" {
		if s.pipelineSvc == nil || s.pipelineRunner == nil {
			http.Error(w, "pipeline service not available", http.StatusServiceUnavailable)
			return
		}
		pipeline, err := s.pipelineSvc.Get(r.Context(), trigger.PipelineID)
		if err != nil {
			http.Error(w, "pipeline not found", http.StatusNotFound)
			return
		}
		go func() {
			_, err := s.pipelineRunner.Start(context.Background(), pipeline, inputs)
			if err != nil {
				slog.Error("webhook: pipeline start failed", "trigger", id, "pipeline", trigger.PipelineID, "err", err)
			} else {
				slog.Info("webhook: pipeline started", "trigger", id, "pipeline", trigger.PipelineID)
			}
		}()
	} else {
		wf, err := s.workflowSvc.Lookup(r.Context(), trigger.WorkflowName)
		if err != nil {
			http.Error(w, "workflow not found", http.StatusNotFound)
			return
		}
		go func() {
			if s.retryExecutor != nil {
				policy := upal.DefaultRetryPolicy()
				events, result, err := s.retryExecutor.ExecuteWithRetry(
					context.Background(), wf, inputs, policy,
					string(upal.TriggerWebhook), trigger.ID,
				)
				if err != nil {
					slog.Error("webhook: execution failed", "trigger", id, "err", err)
					return
				}
				for range events {
				}
				if res, ok := <-result; ok {
					slog.Info("webhook: run completed", "trigger", id, "session", res.SessionID)
				}
			}
		}()
	}

	writeJSONStatus(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"trigger": id,
	})
}

func verifyHMAC(payload []byte, secret, signature string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func mapInputs(payload map[string]any, mapping map[string]string) map[string]any {
	if len(mapping) == 0 {
		return payload
	}
	inputs := make(map[string]any)
	for inputKey, payloadKey := range mapping {
		if val, ok := payload[payloadKey]; ok {
			inputs[inputKey] = val
		}
	}
	return inputs
}
