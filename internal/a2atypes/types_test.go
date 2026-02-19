package a2atypes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPart_TextJSON(t *testing.T) {
	p := TextPart("hello world")

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Part
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != "text" {
		t.Errorf("type = %q, want %q", got.Type, "text")
	}
	if got.Text != "hello world" {
		t.Errorf("text = %q, want %q", got.Text, "hello world")
	}
	if got.MimeType != "text/plain" {
		t.Errorf("mimeType = %q, want %q", got.MimeType, "text/plain")
	}

	// Verify JSON field names
	raw := string(b)
	if !strings.Contains(raw, `"type"`) {
		t.Error("JSON missing 'type' key")
	}
	if !strings.Contains(raw, `"mimeType"`) {
		t.Error("JSON missing 'mimeType' key")
	}
	// Data should be omitted for text parts
	if strings.Contains(raw, `"data"`) {
		t.Error("JSON should not contain 'data' for text part")
	}
}

func TestPart_DataJSON(t *testing.T) {
	payload := map[string]any{"key": "value", "count": float64(42)}
	p := DataPart(payload, "application/json")

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Part
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != "data" {
		t.Errorf("type = %q, want %q", got.Type, "data")
	}
	if got.MimeType != "application/json" {
		t.Errorf("mimeType = %q, want %q", got.MimeType, "application/json")
	}
	if got.Data == nil {
		t.Fatal("data is nil after round-trip")
	}

	// Text should be omitted for data parts
	raw := string(b)
	if strings.Contains(raw, `"text"`) {
		t.Error("JSON should not contain 'text' for data part")
	}

	// Verify the data content survived round-trip
	dataMap, ok := got.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T, want map[string]any", got.Data)
	}
	if dataMap["key"] != "value" {
		t.Errorf("data[key] = %v, want %q", dataMap["key"], "value")
	}
	if dataMap["count"] != float64(42) {
		t.Errorf("data[count] = %v, want 42", dataMap["count"])
	}
}

func TestArtifact_JSON(t *testing.T) {
	a := Artifact{
		Name:        "result",
		Description: "The computation result",
		Parts: []Part{
			TextPart("answer is 42"),
			DataPart(map[string]any{"answer": float64(42)}, "application/json"),
		},
		Index:    0,
		Metadata: map[string]string{"source": "calculator"},
	}

	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Artifact
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != "result" {
		t.Errorf("name = %q, want %q", got.Name, "result")
	}
	if got.Description != "The computation result" {
		t.Errorf("description = %q, want %q", got.Description, "The computation result")
	}
	if got.Index != 0 {
		t.Errorf("index = %d, want 0", got.Index)
	}
	if len(got.Parts) != 2 {
		t.Fatalf("parts count = %d, want 2", len(got.Parts))
	}
	if got.Parts[0].Type != "text" {
		t.Errorf("parts[0].type = %q, want %q", got.Parts[0].Type, "text")
	}
	if got.Parts[1].Type != "data" {
		t.Errorf("parts[1].type = %q, want %q", got.Parts[1].Type, "data")
	}
	if got.Metadata["source"] != "calculator" {
		t.Errorf("metadata[source] = %q, want %q", got.Metadata["source"], "calculator")
	}
}

func TestArtifact_FirstText(t *testing.T) {
	a := Artifact{
		Parts: []Part{
			DataPart(map[string]any{"skip": true}, "application/json"),
			TextPart("first text"),
			TextPart("second text"),
		},
	}

	got := a.FirstText()
	if got != "first text" {
		t.Errorf("FirstText() = %q, want %q", got, "first text")
	}
}

func TestArtifact_FirstText_Empty(t *testing.T) {
	// Artifact with no text parts
	a := Artifact{
		Parts: []Part{
			DataPart(map[string]any{"only": "data"}, "application/json"),
		},
	}

	got := a.FirstText()
	if got != "" {
		t.Errorf("FirstText() = %q, want empty string", got)
	}

	// Artifact with no parts at all
	empty := Artifact{}
	got = empty.FirstText()
	if got != "" {
		t.Errorf("FirstText() on empty = %q, want empty string", got)
	}
}

func TestArtifact_FirstData(t *testing.T) {
	payload := map[string]any{"answer": float64(42)}
	a := Artifact{
		Parts: []Part{
			TextPart("skip this"),
			DataPart(payload, "application/json"),
			DataPart(map[string]any{"second": true}, "application/json"),
		},
	}

	raw := a.FirstData()
	if raw == nil {
		t.Fatal("FirstData() returned nil")
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal FirstData: %v", err)
	}
	if got["answer"] != float64(42) {
		t.Errorf("data[answer] = %v, want 42", got["answer"])
	}

	// Artifact with no data parts
	noData := Artifact{Parts: []Part{TextPart("text only")}}
	if noData.FirstData() != nil {
		t.Error("FirstData() on text-only artifact should return nil")
	}
}

func TestTaskState_Lifecycle(t *testing.T) {
	task := NewTask("ctx-123")

	if task.Status != TaskCreated {
		t.Errorf("status = %q, want %q", task.Status, TaskCreated)
	}
	if task.ContextID != "ctx-123" {
		t.Errorf("contextId = %q, want %q", task.ContextID, "ctx-123")
	}
	if task.ID == "" {
		t.Error("ID should not be empty")
	}
	if !strings.HasPrefix(task.ID, "task-") {
		t.Errorf("ID = %q, should start with 'task-'", task.ID)
	}
	// ID should be unique
	task2 := NewTask("ctx-456")
	if task.ID == task2.ID {
		t.Error("two NewTask calls should produce different IDs")
	}

	// Verify all task states are valid strings
	states := []TaskState{
		TaskCreated, TaskWorking, TaskInputRequired,
		TaskCompleted, TaskFailed, TaskCanceled,
	}
	for _, s := range states {
		if s == "" {
			t.Error("task state should not be empty")
		}
	}
}

func TestTask_JSON(t *testing.T) {
	task := Task{
		ID:        "task-abc123",
		ContextID: "ctx-789",
		Status:    TaskCompleted,
		Messages: []Message{
			{
				Role:  "user",
				Parts: []Part{TextPart("compute 6 * 7")},
			},
			{
				Role:  "agent",
				Parts: []Part{TextPart("The answer is 42.")},
			},
		},
		Artifacts: []Artifact{
			{
				Name:  "result",
				Parts: []Part{TextPart("42")},
				Index: 0,
			},
		},
	}

	b, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Task
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != "task-abc123" {
		t.Errorf("id = %q, want %q", got.ID, "task-abc123")
	}
	if got.ContextID != "ctx-789" {
		t.Errorf("contextId = %q, want %q", got.ContextID, "ctx-789")
	}
	if got.Status != TaskCompleted {
		t.Errorf("status = %q, want %q", got.Status, TaskCompleted)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("messages count = %d, want 2", len(got.Messages))
	}
	if got.Messages[0].Role != "user" {
		t.Errorf("messages[0].role = %q, want %q", got.Messages[0].Role, "user")
	}
	if got.Messages[1].Role != "agent" {
		t.Errorf("messages[1].role = %q, want %q", got.Messages[1].Role, "agent")
	}
	if len(got.Artifacts) != 1 {
		t.Fatalf("artifacts count = %d, want 1", len(got.Artifacts))
	}
	if got.Artifacts[0].Name != "result" {
		t.Errorf("artifacts[0].name = %q, want %q", got.Artifacts[0].Name, "result")
	}

	// Verify JSON uses camelCase field names
	raw := string(b)
	if !strings.Contains(raw, `"contextId"`) {
		t.Error("JSON should use camelCase 'contextId'")
	}
}

func TestMessage_JSON(t *testing.T) {
	msg := Message{
		Role: "user",
		Parts: []Part{
			TextPart("Hello"),
			DataPart(map[string]any{"image_url": "https://example.com/img.png"}, "application/json"),
		},
	}

	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Message
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Role != "user" {
		t.Errorf("role = %q, want %q", got.Role, "user")
	}
	if len(got.Parts) != 2 {
		t.Fatalf("parts count = %d, want 2", len(got.Parts))
	}
	if got.Parts[0].Text != "Hello" {
		t.Errorf("parts[0].text = %q, want %q", got.Parts[0].Text, "Hello")
	}
	if got.Parts[1].Type != "data" {
		t.Errorf("parts[1].type = %q, want %q", got.Parts[1].Type, "data")
	}
}

func TestAgentCard_JSON(t *testing.T) {
	card := AgentCard{
		Name:        "Upal Workflow Agent",
		Description: "Executes DAG-based AI workflows",
		URL:         "http://localhost:8080/.well-known/agent.json",
		Capabilities: Capabilities{
			Streaming:         true,
			PushNotifications: false,
		},
		Skills: []Skill{
			{
				ID:          "workflow-execute",
				Name:        "Execute Workflow",
				Description: "Runs a named workflow with inputs",
				InputModes:  []string{"text/plain", "application/json"},
				OutputModes: []string{"text/plain", "application/json"},
			},
		},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain", "application/json"},
	}

	b, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got AgentCard
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != "Upal Workflow Agent" {
		t.Errorf("name = %q, want %q", got.Name, "Upal Workflow Agent")
	}
	if got.Description != "Executes DAG-based AI workflows" {
		t.Errorf("description = %q, want %q", got.Description, "Executes DAG-based AI workflows")
	}
	if got.URL != "http://localhost:8080/.well-known/agent.json" {
		t.Errorf("url = %q, want %q", got.URL, "http://localhost:8080/.well-known/agent.json")
	}
	if !got.Capabilities.Streaming {
		t.Error("capabilities.streaming should be true")
	}
	if got.Capabilities.PushNotifications {
		t.Error("capabilities.pushNotifications should be false")
	}
	if len(got.Skills) != 1 {
		t.Fatalf("skills count = %d, want 1", len(got.Skills))
	}
	skill := got.Skills[0]
	if skill.ID != "workflow-execute" {
		t.Errorf("skill.id = %q, want %q", skill.ID, "workflow-execute")
	}
	if skill.Name != "Execute Workflow" {
		t.Errorf("skill.name = %q, want %q", skill.Name, "Execute Workflow")
	}
	if len(skill.InputModes) != 2 {
		t.Errorf("skill.inputModes count = %d, want 2", len(skill.InputModes))
	}
	if len(got.DefaultInputModes) != 1 {
		t.Errorf("defaultInputModes count = %d, want 1", len(got.DefaultInputModes))
	}
	if len(got.DefaultOutputModes) != 2 {
		t.Errorf("defaultOutputModes count = %d, want 2", len(got.DefaultOutputModes))
	}

	// Verify camelCase keys in JSON
	raw := string(b)
	for _, key := range []string{"defaultInputModes", "defaultOutputModes", "pushNotifications", "inputModes", "outputModes"} {
		if !strings.Contains(raw, `"`+key+`"`) {
			t.Errorf("JSON missing camelCase key %q", key)
		}
	}
}

func TestJSONRPCRequest_SendMessage(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "a2a.sendMessage",
		Params: SendMessageParams{
			Message: Message{
				Role:  "user",
				Parts: []Part{TextPart("run my workflow")},
			},
			Configuration: &SendMessageConfig{
				AcceptedOutputModes: []string{"text/plain"},
				Blocking:            true,
			},
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got JSONRPCRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want %q", got.JSONRPC, "2.0")
	}
	if got.ID != float64(1) {
		t.Errorf("id = %v, want 1", got.ID)
	}
	if got.Method != "a2a.sendMessage" {
		t.Errorf("method = %q, want %q", got.Method, "a2a.sendMessage")
	}

	// Verify params survived round-trip (will be map after unmarshal)
	paramsMap, ok := got.Params.(map[string]any)
	if !ok {
		t.Fatalf("params type = %T, want map[string]any", got.Params)
	}
	msgMap, ok := paramsMap["message"].(map[string]any)
	if !ok {
		t.Fatal("params.message should be a map")
	}
	if msgMap["role"] != "user" {
		t.Errorf("params.message.role = %v, want %q", msgMap["role"], "user")
	}
}

func TestJSONRPCResponse_Success(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(1),
		Result: Task{
			ID:     "task-abc",
			Status: TaskCompleted,
			Artifacts: []Artifact{
				{
					Name:  "output",
					Parts: []Part{TextPart("done")},
					Index: 0,
				},
			},
		},
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got JSONRPCResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want %q", got.JSONRPC, "2.0")
	}
	if got.ID != float64(1) {
		t.Errorf("id = %v, want 1", got.ID)
	}
	if got.Error != nil {
		t.Error("error should be nil for success response")
	}
	if got.Result == nil {
		t.Fatal("result should not be nil")
	}

	// Error should be omitted from JSON
	raw := string(b)
	if strings.Contains(raw, `"error"`) {
		t.Error("success response JSON should not contain 'error' key")
	}
}

func TestJSONRPCResponse_Error(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(1),
		Error: &JSONRPCError{
			Code:    -32601,
			Message: "Method not found",
			Data:    "unknown method: a2a.fooBar",
		},
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got JSONRPCResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want %q", got.JSONRPC, "2.0")
	}
	if got.Error == nil {
		t.Fatal("error should not be nil")
	}
	if got.Error.Code != -32601 {
		t.Errorf("error.code = %d, want -32601", got.Error.Code)
	}
	if got.Error.Message != "Method not found" {
		t.Errorf("error.message = %q, want %q", got.Error.Message, "Method not found")
	}
	if got.Error.Data != "unknown method: a2a.fooBar" {
		t.Errorf("error.data = %v, want %q", got.Error.Data, "unknown method: a2a.fooBar")
	}
	if got.Result != nil {
		t.Error("result should be nil for error response")
	}

	// Result should be omitted from JSON
	raw := string(b)
	if strings.Contains(raw, `"result"`) {
		t.Error("error response JSON should not contain 'result' key")
	}
}
