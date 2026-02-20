package api

import (
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

// handleWebhook receives an external HTTP POST and triggers a workflow run.
// POST /api/hooks/{id}
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if s.triggerRepo == nil {
		http.Error(w, "triggers not available", http.StatusServiceUnavailable)
		return
	}

	// 1. Look up trigger.
	trigger, err := s.triggerRepo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "trigger not found", http.StatusNotFound)
		return
	}
	if !trigger.Enabled {
		http.Error(w, "trigger is disabled", http.StatusForbidden)
		return
	}

	// 2. Read request body.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// 3. Verify HMAC signature if secret is configured.
	if trigger.Config.Secret != "" {
		signature := r.Header.Get("X-Webhook-Signature")
		if !verifyHMAC(body, trigger.Config.Secret, signature) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// 4. Parse body as JSON and map inputs.
	var payload map[string]any
	if len(body) > 0 {
		json.Unmarshal(body, &payload)
	}

	inputs := mapInputs(payload, trigger.Config.InputMapping)

	// 5. Look up workflow.
	wf, err := s.workflowSvc.Lookup(r.Context(), trigger.WorkflowName)
	if err != nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	// 6. Execute asynchronously.
	go func() {
		if s.retryExecutor != nil {
			policy := upal.DefaultRetryPolicy()
			events, result, err := s.retryExecutor.ExecuteWithRetry(
				r.Context(), wf, inputs, policy,
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

	// 7. Return 202 Accepted immediately.
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"trigger": id,
	})
}

// verifyHMAC checks the HMAC-SHA256 signature of a payload.
func verifyHMAC(payload []byte, secret, signature string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// mapInputs extracts workflow inputs from a webhook payload using the input mapping.
// If no mapping is configured, the entire payload is passed as-is.
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
