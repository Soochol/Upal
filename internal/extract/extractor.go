package extract

import (
	"io"
	"strings"
)

// Extract reads r and returns a text representation of the content.
// Returns ("", nil) for unsupported content types.
func Extract(contentType string, r io.Reader) (string, error) {
	mime := strings.SplitN(contentType, ";", 2)[0]
	mime = strings.TrimSpace(strings.ToLower(mime))

	switch {
	case strings.HasPrefix(mime, "text/"):
		return extractText(r)
	case mime == "application/pdf":
		return extractPDF(r)
	case mime == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return extractDOCX(r)
	case mime == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return extractXLSX(r)
	case strings.HasPrefix(mime, "image/"):
		return extractImage(mime, r)
	default:
		return "", nil
	}
}
