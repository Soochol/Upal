package llmutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soochol/upal/internal/llmutil"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestExtractContentWithAudio(t *testing.T) {
	dir := t.TempDir()
	audioData := []byte("fake-mp3-data")

	resp := &adkmodel.LLMResponse{
		Content: &genai.Content{
			Parts: []*genai.Part{
				{InlineData: &genai.Blob{Data: audioData, MIMEType: "audio/mpeg"}},
			},
		},
	}

	result := llmutil.ExtractContentSavingAudio(resp, dir)

	if !strings.HasSuffix(result, ".mp3") {
		t.Errorf("expected .mp3 path, got %q", result)
	}
	if _, err := os.Stat(result); err != nil {
		t.Errorf("file not found at path %q: %v", result, err)
	}
	data, _ := os.ReadFile(result)
	if string(data) != string(audioData) {
		t.Errorf("file content mismatch")
	}
	_ = filepath.Dir(result)
}

func TestExtractContentSavingAudio_TextPassthrough(t *testing.T) {
	resp := &adkmodel.LLMResponse{
		Content: &genai.Content{
			Parts: []*genai.Part{genai.NewPartFromText("hello world")},
		},
	}
	result := llmutil.ExtractContentSavingAudio(resp, t.TempDir())
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestExtractContentSavingAudio_ImageStillDataURI(t *testing.T) {
	imgData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
	resp := &adkmodel.LLMResponse{
		Content: &genai.Content{
			Parts: []*genai.Part{
				{InlineData: &genai.Blob{Data: imgData, MIMEType: "image/png"}},
			},
		},
	}
	result := llmutil.ExtractContentSavingAudio(resp, t.TempDir())
	if !strings.HasPrefix(result, "data:image/png;base64,") {
		t.Errorf("expected image data URI, got %q", result)
	}
}
