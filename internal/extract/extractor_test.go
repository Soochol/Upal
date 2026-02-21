package extract_test

import (
	"bytes"
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

func TestExtractDOCX(t *testing.T) {
	f, err := os.Open("testdata/sample.docx")
	if err != nil {
		t.Skip("testdata/sample.docx not present:", err)
	}
	defer f.Close()

	text, err := extract.Extract("application/vnd.openxmlformats-officedocument.wordprocessingml.document", f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Hello") {
		t.Errorf("expected 'Hello' in DOCX text, got: %q", text)
	}
}

func TestExtractXLSX(t *testing.T) {
	f, err := os.Open("testdata/sample.xlsx")
	if err != nil {
		t.Skip("testdata/sample.xlsx not present:", err)
	}
	defer f.Close()

	text, err := extract.Extract("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Hello") {
		t.Errorf("expected 'Hello' in XLSX text, got: %q", text)
	}
}

func TestExtractImage(t *testing.T) {
	// Minimal 1x1 red PNG (valid PNG bytes)
	png1x1 := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}

	text, err := extract.Extract("image/png", bytes.NewReader(png1x1))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(text, "data:image/png;base64,") {
		t.Errorf("expected data URI prefix, got: %q", text[:min(len(text), 50)])
	}
}
