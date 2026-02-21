package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/session"
)

// testEnv bundles the fully-wired test server with the repositories that
// back it, so tests can seed data directly when the API does not expose the
// required mutation (e.g. disabling a trigger).
type testEnv struct {
	srv      *Server
	trigRepo repository.TriggerRepository
}

// newFullTestServer creates a Server with every service wired up using
// in-memory stores. This is the primary helper for end-to-end scenario tests
// that exercise cross-component interactions through the HTTP API.
func newFullTestServer() *testEnv {
	sessionSvc := session.InMemoryService()
	wfRepo := repository.NewMemory()
	wfSvc := services.NewWorkflowService(wfRepo, nil, sessionSvc, nil, agents.DefaultRegistry())
	schedRepo := repository.NewMemoryScheduleRepository()
	trigRepo := repository.NewMemoryTriggerRepository()
	runRepo := repository.NewMemoryRunRepository()
	limiter := services.NewConcurrencyLimiter(upal.ConcurrencyLimits{GlobalMax: 10, PerWorkflow: 3})
	runHistorySvc := services.NewRunHistoryService(runRepo)
	retryExec := services.NewRetryExecutor(wfSvc, runHistorySvc)
	schedSvc := services.NewSchedulerService(schedRepo, wfSvc, retryExec, limiter, runHistorySvc)

	srv := NewServer(nil, wfSvc, wfRepo, nil)
	srv.SetSchedulerService(schedSvc)
	srv.SetTriggerRepository(trigRepo)
	srv.SetConcurrencyLimiter(limiter)
	srv.SetRetryExecutor(retryExec)
	srv.SetRunHistoryService(runHistorySvc)
	return &testEnv{srv: srv, trigRepo: trigRepo}
}

// computeHMAC calculates the HMAC-SHA256 hex digest for a payload and secret.
func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// createWorkflowE2E creates a minimal input-to-output workflow via the API.
func createWorkflowE2E(t *testing.T, srv *Server, name string) {
	t.Helper()
	wf := upal.WorkflowDefinition{
		Name:    name,
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "input1", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "output1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "input1", To: "output1"},
		},
	}
	body, _ := json.Marshal(wf)
	req := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("createWorkflowE2E(%s): got status %d, want 201; body: %s", name, w.Code, w.Body.String())
	}
}

// --- E2E Scenario 1: Schedule Lifecycle ---

func TestE2E_ScheduleLifecycle(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	// Step 1: Create a workflow.
	createWorkflowE2E(t, srv, "lifecycle-wf")

	// Step 2: Create a schedule for it.
	schedBody, _ := json.Marshal(map[string]any{
		"workflow_name": "lifecycle-wf",
		"cron_expr":     "*/10 * * * *",
		"enabled":       true,
	})
	createReq := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(schedBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("create schedule: expected 201, got %d: %s", createW.Code, createW.Body.String())
	}

	var created upal.Schedule
	if err := json.NewDecoder(createW.Body).Decode(&created); err != nil {
		t.Fatalf("decode created schedule: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty schedule ID")
	}
	if created.WorkflowName != "lifecycle-wf" {
		t.Errorf("workflow_name: got %q, want %q", created.WorkflowName, "lifecycle-wf")
	}

	// Step 3: List schedules and verify it is present.
	listReq := httptest.NewRequest("GET", "/api/schedules", nil)
	listW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("list schedules: expected 200, got %d", listW.Code)
	}

	var schedules []upal.Schedule
	if err := json.NewDecoder(listW.Body).Decode(&schedules); err != nil {
		t.Fatalf("decode schedule list: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule in list, got %d", len(schedules))
	}
	if schedules[0].ID != created.ID {
		t.Errorf("listed schedule ID: got %q, want %q", schedules[0].ID, created.ID)
	}

	// Step 4: Get schedule by ID and verify data.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get schedule: expected 200, got %d", getW.Code)
	}

	var fetched upal.Schedule
	if err := json.NewDecoder(getW.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched schedule: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("fetched ID: got %q, want %q", fetched.ID, created.ID)
	}
	if fetched.CronExpr != "*/10 * * * *" {
		t.Errorf("fetched cron_expr: got %q, want %q", fetched.CronExpr, "*/10 * * * *")
	}
	if !fetched.Enabled {
		t.Error("expected fetched schedule to be enabled")
	}

	// Step 5: Pause the schedule.
	pauseReq := httptest.NewRequest("POST", "/api/schedules/"+created.ID+"/pause", nil)
	pauseW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(pauseW, pauseReq)

	if pauseW.Code != http.StatusNoContent {
		t.Fatalf("pause: expected 204, got %d: %s", pauseW.Code, pauseW.Body.String())
	}

	// Step 6: Get again and verify enabled=false.
	getReq2 := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW2, getReq2)

	if getW2.Code != http.StatusOK {
		t.Fatalf("get after pause: expected 200, got %d", getW2.Code)
	}

	var paused upal.Schedule
	if err := json.NewDecoder(getW2.Body).Decode(&paused); err != nil {
		t.Fatalf("decode paused schedule: %v", err)
	}
	if paused.Enabled {
		t.Error("expected schedule to be disabled after pause")
	}

	// Step 7: Resume the schedule.
	resumeReq := httptest.NewRequest("POST", "/api/schedules/"+created.ID+"/resume", nil)
	resumeW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(resumeW, resumeReq)

	if resumeW.Code != http.StatusNoContent {
		t.Fatalf("resume: expected 204, got %d: %s", resumeW.Code, resumeW.Body.String())
	}

	// Step 8: Get again and verify enabled=true.
	getReq3 := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW3, getReq3)

	if getW3.Code != http.StatusOK {
		t.Fatalf("get after resume: expected 200, got %d", getW3.Code)
	}

	var resumed upal.Schedule
	if err := json.NewDecoder(getW3.Body).Decode(&resumed); err != nil {
		t.Fatalf("decode resumed schedule: %v", err)
	}
	if !resumed.Enabled {
		t.Error("expected schedule to be enabled after resume")
	}

	// Step 9: Delete the schedule.
	delReq := httptest.NewRequest("DELETE", "/api/schedules/"+created.ID, nil)
	delW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d: %s", delW.Code, delW.Body.String())
	}

	// Step 10: List schedules and verify empty.
	listReq2 := httptest.NewRequest("GET", "/api/schedules", nil)
	listW2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW2, listReq2)

	if listW2.Code != http.StatusOK {
		t.Fatalf("list after delete: expected 200, got %d", listW2.Code)
	}

	var remaining []upal.Schedule
	if err := json.NewDecoder(listW2.Body).Decode(&remaining); err != nil {
		t.Fatalf("decode remaining list: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected 0 schedules after delete, got %d", len(remaining))
	}
}

// --- E2E Scenario 2: Trigger and Webhook Flow ---

func TestE2E_TriggerAndWebhookFlow(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	// Step 1: Create a workflow.
	createWorkflowE2E(t, srv, "webhook-wf")

	// Step 2: Create a trigger for the workflow.
	trigBody := `{"workflow_name": "webhook-wf"}`
	trigReq := httptest.NewRequest("POST", "/api/triggers", strings.NewReader(trigBody))
	trigReq.Header.Set("Content-Type", "application/json")
	trigW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(trigW, trigReq)

	if trigW.Code != http.StatusCreated {
		t.Fatalf("create trigger: expected 201, got %d: %s", trigW.Code, trigW.Body.String())
	}

	// Parse the create response to get trigger details and webhook_url.
	var createResp map[string]json.RawMessage
	if err := json.Unmarshal(trigW.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create trigger response: %v", err)
	}

	if _, ok := createResp["webhook_url"]; !ok {
		t.Fatal("create trigger response missing webhook_url")
	}

	var webhookURL string
	if err := json.Unmarshal(createResp["webhook_url"], &webhookURL); err != nil {
		t.Fatalf("decode webhook_url: %v", err)
	}

	var trigger struct {
		ID     string `json:"id"`
		Config struct {
			Secret string `json:"secret"`
		} `json:"config"`
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(createResp["trigger"], &trigger); err != nil {
		t.Fatalf("decode trigger: %v", err)
	}

	if trigger.ID == "" {
		t.Fatal("trigger ID is empty")
	}
	if !trigger.Enabled {
		t.Error("expected trigger to be enabled")
	}
	if trigger.Config.Secret == "" {
		t.Fatal("expected auto-generated secret")
	}

	expectedURL := "/api/hooks/" + trigger.ID
	if webhookURL != expectedURL {
		t.Errorf("webhook_url: got %q, want %q", webhookURL, expectedURL)
	}

	// Step 3: List triggers for the workflow and verify 1 trigger.
	listReq := httptest.NewRequest("GET", "/api/workflows/webhook-wf/triggers", nil)
	listW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("list triggers: expected 200, got %d", listW.Code)
	}

	var triggers []json.RawMessage
	if err := json.Unmarshal(listW.Body.Bytes(), &triggers); err != nil {
		t.Fatalf("decode trigger list: %v", err)
	}
	if len(triggers) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(triggers))
	}

	// Step 4: Compute HMAC-SHA256 signature and fire the webhook.
	payload := []byte(`{"message":"hello from webhook"}`)
	signature := computeHMAC(payload, trigger.Config.Secret)

	hookReq := httptest.NewRequest("POST", webhookURL, bytes.NewReader(payload))
	hookReq.Header.Set("Content-Type", "application/json")
	hookReq.Header.Set("X-Webhook-Signature", signature)
	hookW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(hookW, hookReq)

	if hookW.Code != http.StatusAccepted {
		t.Fatalf("fire webhook: expected 202, got %d: %s", hookW.Code, hookW.Body.String())
	}

	var hookResp map[string]string
	if err := json.Unmarshal(hookW.Body.Bytes(), &hookResp); err != nil {
		t.Fatalf("decode webhook response: %v", err)
	}
	if hookResp["status"] != "accepted" {
		t.Errorf("webhook status: got %q, want %q", hookResp["status"], "accepted")
	}
	if hookResp["trigger"] != trigger.ID {
		t.Errorf("webhook trigger: got %q, want %q", hookResp["trigger"], trigger.ID)
	}

	// Step 5: Delete the trigger.
	delReq := httptest.NewRequest("DELETE", "/api/triggers/"+trigger.ID, nil)
	delW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Fatalf("delete trigger: expected 204, got %d: %s", delW.Code, delW.Body.String())
	}

	// Step 6: List triggers and verify empty.
	listReq2 := httptest.NewRequest("GET", "/api/workflows/webhook-wf/triggers", nil)
	listW2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW2, listReq2)

	if listW2.Code != http.StatusOK {
		t.Fatalf("list triggers after delete: expected 200, got %d", listW2.Code)
	}

	var remaining []json.RawMessage
	if err := json.Unmarshal(listW2.Body.Bytes(), &remaining); err != nil {
		t.Fatalf("decode remaining triggers: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected 0 triggers after delete, got %d", len(remaining))
	}
}

// --- E2E Scenario 3: Multiple Schedules Same Workflow ---

func TestE2E_MultipleSchedulesSameWorkflow(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	// Create a workflow.
	createWorkflowE2E(t, srv, "multi-sched-wf")

	// Create 3 schedules with different cron expressions.
	cronExprs := []string{"*/5 * * * *", "0 * * * *", "30 12 * * *"}
	createdIDs := make([]string, 0, 3)

	for _, cron := range cronExprs {
		body, _ := json.Marshal(map[string]any{
			"workflow_name": "multi-sched-wf",
			"cron_expr":     cron,
			"enabled":       true,
		})
		req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("create schedule (cron=%s): expected 201, got %d: %s", cron, w.Code, w.Body.String())
		}

		var s upal.Schedule
		if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
			t.Fatalf("decode schedule: %v", err)
		}
		createdIDs = append(createdIDs, s.ID)
	}

	// List schedules and verify all 3 are present.
	listReq := httptest.NewRequest("GET", "/api/schedules", nil)
	listW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("list schedules: expected 200, got %d", listW.Code)
	}

	var schedules []upal.Schedule
	if err := json.NewDecoder(listW.Body).Decode(&schedules); err != nil {
		t.Fatalf("decode schedule list: %v", err)
	}
	if len(schedules) != 3 {
		t.Fatalf("expected 3 schedules, got %d", len(schedules))
	}

	// Delete the first schedule.
	delReq := httptest.NewRequest("DELETE", "/api/schedules/"+createdIDs[0], nil)
	delW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Fatalf("delete schedule: expected 204, got %d: %s", delW.Code, delW.Body.String())
	}

	// List again and verify 2 remain.
	listReq2 := httptest.NewRequest("GET", "/api/schedules", nil)
	listW2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW2, listReq2)

	if listW2.Code != http.StatusOK {
		t.Fatalf("list after delete: expected 200, got %d", listW2.Code)
	}

	var remaining []upal.Schedule
	if err := json.NewDecoder(listW2.Body).Decode(&remaining); err != nil {
		t.Fatalf("decode remaining: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 schedules after deleting one, got %d", len(remaining))
	}

	// Verify the deleted one is not in the list.
	for _, s := range remaining {
		if s.ID == createdIDs[0] {
			t.Errorf("deleted schedule %q still present in list", createdIDs[0])
		}
	}
}

// --- E2E Scenario 4: Scheduler Stats ---

func TestE2E_SchedulerStats(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	// GET /api/scheduler/stats with a freshly-created limiter.
	req := httptest.NewRequest("GET", "/api/scheduler/stats", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("scheduler stats: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode stats response: %v", err)
	}

	concRaw, ok := resp["concurrency"]
	if !ok {
		t.Fatal("stats response missing 'concurrency' field")
	}

	var stats struct {
		ActiveRuns  int `json:"active_runs"`
		GlobalMax   int `json:"global_max"`
		PerWorkflow int `json:"per_workflow"`
	}
	if err := json.Unmarshal(concRaw, &stats); err != nil {
		t.Fatalf("decode concurrency stats: %v", err)
	}

	if stats.GlobalMax != 10 {
		t.Errorf("global_max: got %d, want 10", stats.GlobalMax)
	}
	if stats.PerWorkflow != 3 {
		t.Errorf("per_workflow: got %d, want 3", stats.PerWorkflow)
	}
	if stats.ActiveRuns != 0 {
		t.Errorf("active_runs: got %d, want 0", stats.ActiveRuns)
	}
}

// --- E2E Scenario 5: Webhook Without Workflow ---

func TestE2E_WebhookWithoutWorkflow(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	// Seed a trigger directly into the repo for a workflow that does not exist.
	// The create trigger API does not validate workflow existence, so we create
	// the trigger via the API and then fire the webhook without creating the workflow.
	trigger := &upal.Trigger{
		ID:           upal.GenerateID("trig"),
		WorkflowName: "nonexistent-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{}, // no secret, so HMAC is skipped
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := env.trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("seed trigger: %v", err)
	}

	// Fire the webhook. The handler should look up the workflow and return 404.
	payload := []byte(`{"data":"test"}`)
	req := httptest.NewRequest("POST", "/api/hooks/"+trigger.ID, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("webhook without workflow: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- E2E Scenario 6: Webhook Disabled Trigger ---

func TestE2E_WebhookDisabledTrigger(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	// Create the workflow so we can isolate the "disabled" check.
	createWorkflowE2E(t, srv, "disabled-trig-wf")

	// Seed a disabled trigger directly into the repo.
	// There is no API to disable a trigger, so we seed it directly.
	trigger := &upal.Trigger{
		ID:           upal.GenerateID("trig"),
		WorkflowName: "disabled-trig-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{},
		Enabled:      false,
		CreatedAt:    time.Now(),
	}
	if err := env.trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("seed disabled trigger: %v", err)
	}

	// Fire the webhook. The handler should reject it with 403.
	payload := []byte(`{"data":"test"}`)
	req := httptest.NewRequest("POST", "/api/hooks/"+trigger.ID, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("webhook disabled trigger: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// --- E2E Scenario 7: Schedule Trigger → Run Record Integration ---

func TestE2E_ScheduleRunRecordIntegration(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	// Step 1: Create a workflow.
	createWorkflowE2E(t, srv, "run-record-wf")

	// Step 2: Create a schedule.
	schedBody, _ := json.Marshal(map[string]any{
		"workflow_name": "run-record-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	createReq := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(schedBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("create schedule: expected 201, got %d: %s", createW.Code, createW.Body.String())
	}

	var sched upal.Schedule
	if err := json.NewDecoder(createW.Body).Decode(&sched); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}

	// Step 3: Trigger the schedule immediately via POST /api/schedules/{id}/trigger.
	trigReq := httptest.NewRequest("POST", "/api/schedules/"+sched.ID+"/trigger", nil)
	trigW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(trigW, trigReq)

	if trigW.Code != http.StatusOK {
		t.Fatalf("trigger: expected 200, got %d: %s", trigW.Code, trigW.Body.String())
	}

	var trigResp map[string]string
	if err := json.Unmarshal(trigW.Body.Bytes(), &trigResp); err != nil {
		t.Fatalf("decode trigger response: %v", err)
	}
	if trigResp["status"] != "completed" {
		t.Errorf("trigger status: got %q, want %q", trigResp["status"], "completed")
	}
	if trigResp["schedule"] != sched.ID {
		t.Errorf("trigger schedule: got %q, want %q", trigResp["schedule"], sched.ID)
	}

	// Step 4: Verify the run record in /api/runs.
	runsReq := httptest.NewRequest("GET", "/api/runs?limit=10", nil)
	runsW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(runsW, runsReq)

	if runsW.Code != http.StatusOK {
		t.Fatalf("list runs: expected 200, got %d: %s", runsW.Code, runsW.Body.String())
	}

	var runsResp struct {
		Runs  []upal.RunRecord `json:"runs"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(runsW.Body.Bytes(), &runsResp); err != nil {
		t.Fatalf("decode runs: %v", err)
	}
	if runsResp.Total == 0 {
		t.Fatal("expected at least 1 run record, got 0")
	}

	// Find the run triggered by our schedule.
	var found *upal.RunRecord
	for i := range runsResp.Runs {
		r := &runsResp.Runs[i]
		if r.TriggerRef == sched.ID && r.TriggerType == "cron" {
			found = r
			break
		}
	}
	if found == nil {
		t.Fatalf("no run record with trigger_type=cron and trigger_ref=%s found", sched.ID)
	}
	if found.WorkflowName != "run-record-wf" {
		t.Errorf("run workflow_name: got %q, want %q", found.WorkflowName, "run-record-wf")
	}

	// Step 5: Verify schedule's LastRunAt was updated.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+sched.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get schedule: expected 200, got %d", getW.Code)
	}

	var afterTrigger upal.Schedule
	if err := json.NewDecoder(getW.Body).Decode(&afterTrigger); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	if afterTrigger.LastRunAt == nil {
		t.Error("expected LastRunAt to be set after trigger")
	}
}

// --- E2E Scenario 8: Trigger Non-existent Schedule ---

func TestE2E_TriggerNonexistentSchedule(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	req := httptest.NewRequest("POST", "/api/schedules/sched-ghost/trigger", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("trigger nonexistent: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- E2E Scenario 9: Trigger Without Scheduler Service ---

func TestE2E_TriggerNoScheduler(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("POST", "/api/schedules/some-id/trigger", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("trigger no scheduler: expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================
// Phase 1 — Tier 1: Data Integrity (RetryPolicy, Timezone, Inputs)
// ============================================================

// --- E2E Scenario 7: Schedule with Retry Policy ---

func TestE2E_ScheduleWithRetryPolicy(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	createWorkflowE2E(t, srv, "retry-policy-wf")

	// Step 1: Create schedule with explicit retry policy.
	schedBody, _ := json.Marshal(map[string]any{
		"workflow_name": "retry-policy-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
		"retry_policy": map[string]any{
			"max_retries":    5,
			"initial_delay":  2000000000, // 2s in nanoseconds (time.Duration)
			"max_delay":      60000000000,
			"backoff_factor": 1.5,
		},
	})
	req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(schedBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create schedule: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode created schedule: %v", err)
	}
	if created.RetryPolicy == nil {
		t.Fatal("expected retry_policy to be set, got nil")
	}
	if created.RetryPolicy.MaxRetries != 5 {
		t.Errorf("max_retries: got %d, want 5", created.RetryPolicy.MaxRetries)
	}
	if created.RetryPolicy.BackoffFactor != 1.5 {
		t.Errorf("backoff_factor: got %f, want 1.5", created.RetryPolicy.BackoffFactor)
	}

	// Step 2: GET and verify retry policy roundtrip.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get schedule: expected 200, got %d", getW.Code)
	}

	var fetched upal.Schedule
	if err := json.NewDecoder(getW.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched: %v", err)
	}
	if fetched.RetryPolicy == nil {
		t.Fatal("fetched retry_policy is nil")
	}
	if fetched.RetryPolicy.MaxRetries != 5 {
		t.Errorf("fetched max_retries: got %d, want 5", fetched.RetryPolicy.MaxRetries)
	}
	if fetched.RetryPolicy.BackoffFactor != 1.5 {
		t.Errorf("fetched backoff_factor: got %f, want 1.5", fetched.RetryPolicy.BackoffFactor)
	}

	// Step 3: Update retry policy via PUT.
	updateBody, _ := json.Marshal(map[string]any{
		"workflow_name": "retry-policy-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
		"retry_policy": map[string]any{
			"max_retries":    10,
			"initial_delay":  5000000000,
			"max_delay":      120000000000,
			"backoff_factor": 3.0,
		},
	})
	putReq := httptest.NewRequest("PUT", "/api/schedules/"+created.ID, bytes.NewReader(updateBody))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("update schedule: expected 200, got %d: %s", putW.Code, putW.Body.String())
	}

	var updated upal.Schedule
	if err := json.NewDecoder(putW.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated: %v", err)
	}
	if updated.RetryPolicy == nil {
		t.Fatal("updated retry_policy is nil")
	}
	if updated.RetryPolicy.MaxRetries != 10 {
		t.Errorf("updated max_retries: got %d, want 10", updated.RetryPolicy.MaxRetries)
	}
	if updated.RetryPolicy.BackoffFactor != 3.0 {
		t.Errorf("updated backoff_factor: got %f, want 3.0", updated.RetryPolicy.BackoffFactor)
	}

	// Step 4: GET after update to confirm persistence.
	getReq2 := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW2, getReq2)

	if getW2.Code != http.StatusOK {
		t.Fatalf("get after update: expected 200, got %d", getW2.Code)
	}

	var final upal.Schedule
	if err := json.NewDecoder(getW2.Body).Decode(&final); err != nil {
		t.Fatalf("decode final: %v", err)
	}
	if final.RetryPolicy.MaxRetries != 10 {
		t.Errorf("final max_retries: got %d, want 10", final.RetryPolicy.MaxRetries)
	}
}

// --- E2E Scenario 8: Schedule with Timezone ---

func TestE2E_ScheduleWithTimezone(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	createWorkflowE2E(t, srv, "tz-wf")

	// Step 1: Create with explicit timezone.
	body1, _ := json.Marshal(map[string]any{
		"workflow_name": "tz-wf",
		"cron_expr":     "0 9 * * *",
		"enabled":       true,
		"timezone":      "Asia/Seoul",
	})
	req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create with timezone: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var withTz upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&withTz); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if withTz.Timezone != "Asia/Seoul" {
		t.Errorf("timezone: got %q, want %q", withTz.Timezone, "Asia/Seoul")
	}

	// Step 2: GET and verify timezone persists.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+withTz.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getW.Code)
	}

	var fetched upal.Schedule
	if err := json.NewDecoder(getW.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched: %v", err)
	}
	if fetched.Timezone != "Asia/Seoul" {
		t.Errorf("fetched timezone: got %q, want %q", fetched.Timezone, "Asia/Seoul")
	}

	// Step 3: Create without timezone → default to "UTC".
	body2, _ := json.Marshal(map[string]any{
		"workflow_name": "tz-wf",
		"cron_expr":     "0 12 * * *",
		"enabled":       true,
	})
	req2 := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("create without timezone: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var noTz upal.Schedule
	if err := json.NewDecoder(w2.Body).Decode(&noTz); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if noTz.Timezone != "UTC" {
		t.Errorf("default timezone: got %q, want %q", noTz.Timezone, "UTC")
	}
}

// --- E2E Scenario 9: Schedule with Inputs ---

func TestE2E_ScheduleWithInputs(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	createWorkflowE2E(t, srv, "inputs-wf")

	// Create schedule with inputs.
	body, _ := json.Marshal(map[string]any{
		"workflow_name": "inputs-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
		"inputs": map[string]any{
			"query":   "hello world",
			"count":   42,
			"verbose": true,
		},
	})
	req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Inputs == nil {
		t.Fatal("expected inputs to be set, got nil")
	}
	if created.Inputs["query"] != "hello world" {
		t.Errorf("inputs.query: got %v, want %q", created.Inputs["query"], "hello world")
	}

	// GET and verify inputs roundtrip.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getW.Code)
	}

	var fetched upal.Schedule
	if err := json.NewDecoder(getW.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched: %v", err)
	}
	if fetched.Inputs == nil {
		t.Fatal("fetched inputs is nil")
	}
	if fetched.Inputs["query"] != "hello world" {
		t.Errorf("fetched inputs.query: got %v, want %q", fetched.Inputs["query"], "hello world")
	}
	// JSON numbers decode as float64.
	if fetched.Inputs["count"] != float64(42) {
		t.Errorf("fetched inputs.count: got %v, want 42", fetched.Inputs["count"])
	}
	if fetched.Inputs["verbose"] != true {
		t.Errorf("fetched inputs.verbose: got %v, want true", fetched.Inputs["verbose"])
	}
}

// ============================================================
// Phase 2 — Tier 2: Integration & Edge Cases
// ============================================================

// --- E2E Scenario 10: Update Non-existent Schedule ---

func TestE2E_UpdateSchedule_NotFound(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	body, _ := json.Marshal(map[string]any{
		"workflow_name": "nope-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	req := httptest.NewRequest("PUT", "/api/schedules/sched-nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("update nonexistent: expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- E2E Scenario 11: Update CronExpr → NextRunAt Recalculated ---

func TestE2E_UpdateSchedule_NextRunAtRecalculation(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	createWorkflowE2E(t, srv, "nextrun-wf")

	// Create with "every hour" cron.
	createBody, _ := json.Marshal(map[string]any{
		"workflow_name": "nextrun-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	createReq := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createW.Code, createW.Body.String())
	}

	var created upal.Schedule
	if err := json.NewDecoder(createW.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	originalNextRun := created.NextRunAt

	// Update cron to "every 30 minutes".
	updateBody, _ := json.Marshal(map[string]any{
		"workflow_name": "nextrun-wf",
		"cron_expr":     "*/30 * * * *",
		"enabled":       true,
	})
	putReq := httptest.NewRequest("PUT", "/api/schedules/"+created.ID, bytes.NewReader(updateBody))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", putW.Code, putW.Body.String())
	}

	// GET and verify NextRunAt changed.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getW.Code)
	}

	var updated upal.Schedule
	if err := json.NewDecoder(getW.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.CronExpr != "*/30 * * * *" {
		t.Errorf("cron_expr: got %q, want %q", updated.CronExpr, "*/30 * * * *")
	}
	// "*/30 * * * *" should produce a NextRunAt <= the original "0 * * * *" NextRunAt.
	if updated.NextRunAt.After(originalNextRun) {
		t.Errorf("expected updated NextRunAt (%v) to be at or before original (%v)",
			updated.NextRunAt, originalNextRun)
	}
}

// --- E2E Scenario 12: Six-field Cron Expression ---

func TestE2E_SixFieldCronExpression(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	createWorkflowE2E(t, srv, "six-field-wf")

	body, _ := json.Marshal(map[string]any{
		"workflow_name": "six-field-wf",
		"cron_expr":     "0 */5 * * * *", // every 5 minutes, 6-field with seconds
		"enabled":       true,
	})
	req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create 6-field cron: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.CronExpr != "0 */5 * * * *" {
		t.Errorf("cron_expr: got %q, want %q", created.CronExpr, "0 */5 * * * *")
	}
	if created.NextRunAt.IsZero() {
		t.Error("expected NextRunAt to be computed for 6-field cron")
	}

	// GET and verify.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getW.Code)
	}

	var fetched upal.Schedule
	if err := json.NewDecoder(getW.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched: %v", err)
	}
	if fetched.CronExpr != "0 */5 * * * *" {
		t.Errorf("fetched cron_expr: got %q, want %q", fetched.CronExpr, "0 */5 * * * *")
	}
}

// --- E2E Scenario 13: Webhook HMAC Validation E2E ---

func TestE2E_WebhookHMACValidation(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	createWorkflowE2E(t, srv, "hmac-wf")

	// Create trigger (auto-generated secret).
	trigBody := `{"workflow_name": "hmac-wf"}`
	trigReq := httptest.NewRequest("POST", "/api/triggers", strings.NewReader(trigBody))
	trigReq.Header.Set("Content-Type", "application/json")
	trigW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(trigW, trigReq)

	if trigW.Code != http.StatusCreated {
		t.Fatalf("create trigger: expected 201, got %d: %s", trigW.Code, trigW.Body.String())
	}

	var createResp map[string]json.RawMessage
	if err := json.Unmarshal(trigW.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	var trigger struct {
		ID     string `json:"id"`
		Config struct {
			Secret string `json:"secret"`
		} `json:"config"`
	}
	if err := json.Unmarshal(createResp["trigger"], &trigger); err != nil {
		t.Fatalf("decode trigger: %v", err)
	}

	var webhookURL string
	if err := json.Unmarshal(createResp["webhook_url"], &webhookURL); err != nil {
		t.Fatalf("decode webhook_url: %v", err)
	}

	payload := []byte(`{"event":"test","value":123}`)

	// Sub-test 1: Valid signature → 202.
	t.Run("valid_signature", func(t *testing.T) {
		sig := computeHMAC(payload, trigger.Config.Secret)
		req := httptest.NewRequest("POST", webhookURL, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Signature", sig)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("valid sig: expected 202, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Sub-test 2: Wrong signature → 401.
	t.Run("wrong_signature", func(t *testing.T) {
		wrongSig := computeHMAC(payload, "wrong-secret")
		req := httptest.NewRequest("POST", webhookURL, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Signature", wrongSig)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("wrong sig: expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Sub-test 3: Missing signature → 401.
	t.Run("missing_signature", func(t *testing.T) {
		req := httptest.NewRequest("POST", webhookURL, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		// No X-Webhook-Signature header.
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("missing sig: expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- E2E Scenario 14: Schedule + Trigger Coexistence ---

func TestE2E_ScheduleAndTriggerCoexistence(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	createWorkflowE2E(t, srv, "coexist-wf")

	// Step 1: Create a schedule.
	schedBody, _ := json.Marshal(map[string]any{
		"workflow_name": "coexist-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	schedReq := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(schedBody))
	schedReq.Header.Set("Content-Type", "application/json")
	schedW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(schedW, schedReq)

	if schedW.Code != http.StatusCreated {
		t.Fatalf("create schedule: expected 201, got %d: %s", schedW.Code, schedW.Body.String())
	}

	var sched upal.Schedule
	if err := json.NewDecoder(schedW.Body).Decode(&sched); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}

	// Step 2: Create a trigger for the same workflow.
	trigBody := `{"workflow_name": "coexist-wf"}`
	trigReq := httptest.NewRequest("POST", "/api/triggers", strings.NewReader(trigBody))
	trigReq.Header.Set("Content-Type", "application/json")
	trigW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(trigW, trigReq)

	if trigW.Code != http.StatusCreated {
		t.Fatalf("create trigger: expected 201, got %d: %s", trigW.Code, trigW.Body.String())
	}

	var trigResp map[string]json.RawMessage
	if err := json.Unmarshal(trigW.Body.Bytes(), &trigResp); err != nil {
		t.Fatalf("decode trigger response: %v", err)
	}
	var trigger struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(trigResp["trigger"], &trigger); err != nil {
		t.Fatalf("decode trigger: %v", err)
	}

	// Step 3: Delete the schedule. Trigger must still exist.
	delSchedReq := httptest.NewRequest("DELETE", "/api/schedules/"+sched.ID, nil)
	delSchedW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delSchedW, delSchedReq)

	if delSchedW.Code != http.StatusNoContent {
		t.Fatalf("delete schedule: expected 204, got %d: %s", delSchedW.Code, delSchedW.Body.String())
	}

	// Trigger still exists.
	listTrigReq := httptest.NewRequest("GET", "/api/workflows/coexist-wf/triggers", nil)
	listTrigW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listTrigW, listTrigReq)

	if listTrigW.Code != http.StatusOK {
		t.Fatalf("list triggers: expected 200, got %d", listTrigW.Code)
	}

	var triggers []json.RawMessage
	if err := json.Unmarshal(listTrigW.Body.Bytes(), &triggers); err != nil {
		t.Fatalf("decode triggers: %v", err)
	}
	if len(triggers) != 1 {
		t.Fatalf("expected 1 trigger after schedule delete, got %d", len(triggers))
	}

	// Schedule is gone.
	listSchedReq := httptest.NewRequest("GET", "/api/schedules", nil)
	listSchedW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listSchedW, listSchedReq)

	var schedules []upal.Schedule
	if err := json.NewDecoder(listSchedW.Body).Decode(&schedules); err != nil {
		t.Fatalf("decode schedules: %v", err)
	}
	if len(schedules) != 0 {
		t.Fatalf("expected 0 schedules after delete, got %d", len(schedules))
	}

	// Step 4: Delete the trigger. Should have no side effects.
	delTrigReq := httptest.NewRequest("DELETE", "/api/triggers/"+trigger.ID, nil)
	delTrigW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delTrigW, delTrigReq)

	if delTrigW.Code != http.StatusNoContent {
		t.Fatalf("delete trigger: expected 204, got %d: %s", delTrigW.Code, delTrigW.Body.String())
	}
}

// --- E2E Scenario 15: Stats After Schedule Changes ---

func TestE2E_StatsAfterScheduleChanges(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	// Baseline stats.
	statsReq := httptest.NewRequest("GET", "/api/scheduler/stats", nil)
	statsW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(statsW, statsReq)

	if statsW.Code != http.StatusOK {
		t.Fatalf("stats: expected 200, got %d", statsW.Code)
	}

	var baseResp map[string]json.RawMessage
	if err := json.Unmarshal(statsW.Body.Bytes(), &baseResp); err != nil {
		t.Fatalf("decode baseline stats: %v", err)
	}
	var baseStats struct {
		ActiveRuns int `json:"active_runs"`
	}
	if err := json.Unmarshal(baseResp["concurrency"], &baseStats); err != nil {
		t.Fatalf("decode concurrency: %v", err)
	}
	if baseStats.ActiveRuns != 0 {
		t.Errorf("baseline active_runs: got %d, want 0", baseStats.ActiveRuns)
	}

	// Create 3 schedules.
	createWorkflowE2E(t, srv, "stats-wf")
	cronExprs := []string{"0 * * * *", "30 * * * *", "0 12 * * *"}
	for _, cronExpr := range cronExprs {
		body, _ := json.Marshal(map[string]any{
			"workflow_name": "stats-wf",
			"cron_expr":     cronExpr,
			"enabled":       true,
		})
		req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create schedule (%s): expected 201, got %d", cronExpr, w.Code)
		}
	}

	// Verify schedules exist via list.
	listReq := httptest.NewRequest("GET", "/api/schedules", nil)
	listW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW, listReq)

	var schedList []upal.Schedule
	if err := json.NewDecoder(listW.Body).Decode(&schedList); err != nil {
		t.Fatalf("decode schedule list: %v", err)
	}
	if len(schedList) != 3 {
		t.Fatalf("expected 3 schedules, got %d", len(schedList))
	}

	// Stats after creation — active_runs still 0 (no execution triggered).
	statsReq2 := httptest.NewRequest("GET", "/api/scheduler/stats", nil)
	statsW2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(statsW2, statsReq2)

	if statsW2.Code != http.StatusOK {
		t.Fatalf("stats after creation: expected 200, got %d", statsW2.Code)
	}

	var afterResp map[string]json.RawMessage
	if err := json.Unmarshal(statsW2.Body.Bytes(), &afterResp); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	var afterStats struct {
		ActiveRuns  int `json:"active_runs"`
		GlobalMax   int `json:"global_max"`
		PerWorkflow int `json:"per_workflow"`
	}
	if err := json.Unmarshal(afterResp["concurrency"], &afterStats); err != nil {
		t.Fatalf("decode concurrency: %v", err)
	}
	if afterStats.ActiveRuns != 0 {
		t.Errorf("active_runs after creation: got %d, want 0", afterStats.ActiveRuns)
	}
	if afterStats.GlobalMax != 10 {
		t.Errorf("global_max: got %d, want 10", afterStats.GlobalMax)
	}
	if afterStats.PerWorkflow != 3 {
		t.Errorf("per_workflow: got %d, want 3", afterStats.PerWorkflow)
	}
}

// ============================================================
// Phase 3 — Tier 3: Defensive Coverage
// ============================================================

// --- E2E Scenario 16: Pause Non-existent Schedule ---

func TestE2E_PauseNonexistentSchedule(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	req := httptest.NewRequest("POST", "/api/schedules/sched-ghost/pause", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("pause nonexistent: expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- E2E Scenario 17: Resume Non-existent Schedule ---

func TestE2E_ResumeNonexistentSchedule(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	req := httptest.NewRequest("POST", "/api/schedules/sched-ghost/resume", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("resume nonexistent: expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- E2E Scenario 18: Delete Non-existent Schedule ---

func TestE2E_DeleteNonexistentSchedule(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	req := httptest.NewRequest("DELETE", "/api/schedules/sched-ghost", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("delete nonexistent: expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- E2E Scenario 19: Delete Paused Schedule ---

func TestE2E_DeletePausedSchedule(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	createWorkflowE2E(t, srv, "del-paused-wf")

	// Create and pause.
	body, _ := json.Marshal(map[string]any{
		"workflow_name": "del-paused-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       true,
	})
	createReq := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", createW.Code)
	}

	var sched upal.Schedule
	if err := json.NewDecoder(createW.Body).Decode(&sched); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Pause.
	pauseReq := httptest.NewRequest("POST", "/api/schedules/"+sched.ID+"/pause", nil)
	pauseW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(pauseW, pauseReq)

	if pauseW.Code != http.StatusNoContent {
		t.Fatalf("pause: expected 204, got %d", pauseW.Code)
	}

	// Delete paused schedule.
	delReq := httptest.NewRequest("DELETE", "/api/schedules/"+sched.ID, nil)
	delW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Fatalf("delete paused: expected 204, got %d: %s", delW.Code, delW.Body.String())
	}

	// Verify gone.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+sched.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", getW.Code)
	}
}

// --- E2E Scenario 20: Create Disabled Schedule ---

func TestE2E_CreateDisabledSchedule(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	createWorkflowE2E(t, srv, "disabled-wf")

	body, _ := json.Marshal(map[string]any{
		"workflow_name": "disabled-wf",
		"cron_expr":     "0 * * * *",
		"enabled":       false,
	})
	req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create disabled: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created upal.Schedule
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Enabled {
		t.Error("expected schedule to be disabled")
	}
	if created.ID == "" {
		t.Error("expected non-empty ID")
	}

	// GET verifies it's stored but disabled.
	getReq := httptest.NewRequest("GET", "/api/schedules/"+created.ID, nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get disabled: expected 200, got %d", getW.Code)
	}

	var fetched upal.Schedule
	if err := json.NewDecoder(getW.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched: %v", err)
	}
	if fetched.Enabled {
		t.Error("fetched schedule should be disabled")
	}
}

// --- E2E Scenario 21: Multi-Workflow Schedules ---

func TestE2E_MultiWorkflowSchedules(t *testing.T) {
	env := newFullTestServer()
	srv := env.srv

	// Create 3 different workflows.
	workflows := []string{"multi-wf-a", "multi-wf-b", "multi-wf-c"}
	for _, name := range workflows {
		createWorkflowE2E(t, srv, name)
	}

	// Create one schedule per workflow.
	var allIDs []string
	for i, wf := range workflows {
		cronExprs := []string{"*/5 * * * *", "*/10 * * * *", "*/15 * * * *"}
		body, _ := json.Marshal(map[string]any{
			"workflow_name": wf,
			"cron_expr":     cronExprs[i],
			"enabled":       true,
		})
		req := httptest.NewRequest("POST", "/api/schedules", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("create schedule for %s: expected 201, got %d", wf, w.Code)
		}

		var sched upal.Schedule
		if err := json.NewDecoder(w.Body).Decode(&sched); err != nil {
			t.Fatalf("decode: %v", err)
		}
		allIDs = append(allIDs, sched.ID)
	}

	// List all schedules — should be 3.
	listReq := httptest.NewRequest("GET", "/api/schedules", nil)
	listW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW, listReq)

	var all []upal.Schedule
	if err := json.NewDecoder(listW.Body).Decode(&all); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 schedules across workflows, got %d", len(all))
	}

	// Verify each schedule has a distinct workflow name.
	names := make(map[string]bool)
	for _, s := range all {
		names[s.WorkflowName] = true
	}
	if len(names) != 3 {
		t.Errorf("expected 3 distinct workflow names, got %d: %v", len(names), names)
	}

	// Delete one and verify others remain.
	delReq := httptest.NewRequest("DELETE", "/api/schedules/"+allIDs[1], nil)
	delW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delW, delReq)

	if delW.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", delW.Code)
	}

	listReq2 := httptest.NewRequest("GET", "/api/schedules", nil)
	listW2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW2, listReq2)

	var remaining []upal.Schedule
	if err := json.NewDecoder(listW2.Body).Decode(&remaining); err != nil {
		t.Fatalf("decode remaining: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 schedules after delete, got %d", len(remaining))
	}
	for _, s := range remaining {
		if s.ID == allIDs[1] {
			t.Errorf("deleted schedule %q still present", allIDs[1])
		}
	}
}
