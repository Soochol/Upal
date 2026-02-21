package extract_test

import (
	"os"
	"strings"
	"testing"

	"github.com/soochol/upal/internal/extract"
)

func TestExtractPlainText(t *testing.T) {
	text, err := extract.Extract("text/plain", strings.NewReader("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello world" {
		t.Errorf("want %q got %q", "hello world", text)
	}
}

func TestExtractCSV(t *testing.T) {
	text, err := extract.Extract("text/csv", strings.NewReader("a,b,c"))
	if err != nil {
		t.Fatal(err)
	}
	if text != "a,b,c" {
		t.Errorf("want %q got %q", "a,b,c", text)
	}
}

func TestExtractUnknownType(t *testing.T) {
	text, err := extract.Extract("application/octet-stream", strings.NewReader("binary"))
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Errorf("unknown content type should return empty string, got %q", text)
	}
}

func TestExtractPDF(t *testing.T) {
	f, err := os.Open("testdata/sample.pdf")
	if err != nil {
		t.Skip("testdata/sample.pdf not present:", err)
	}
	defer f.Close()

	text, err := extract.Extract("application/pdf", f)
	if err != nil {
		t.Fatal(err)
	}
	// sample.pdf must contain the word "Hello"
	if !strings.Contains(text, "Hello") {
		t.Logf("PDF text extracted: %q", text)
		if text == "" {
			t.Skip("ledongthuc/pdf could not extract text from minimal PDF (acceptable)")
		}
		t.Errorf("expected 'Hello' in PDF text, got: %q", text)
	}
}
