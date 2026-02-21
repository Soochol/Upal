package extract

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// extractDOCX reads a DOCX file (ZIP+XML) and extracts paragraph text.
func extractDOCX(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read docx: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open docx zip: %w", err)
	}

	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		return parseDOCXXML(rc)
	}
	return "", fmt.Errorf("word/document.xml not found in docx")
}

func parseDOCXXML(r io.Reader) (string, error) {
	var sb strings.Builder
	decoder := xml.NewDecoder(r)
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sb.String(), nil
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "t":
			var content struct {
				Text string `xml:",chardata"`
			}
			if err := decoder.DecodeElement(&content, &se); err == nil {
				sb.WriteString(content.Text)
			}
		case "p":
			sb.WriteString("\n")
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

// extractXLSX reads an XLSX file and returns all cell values tab/newline separated.
func extractXLSX(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read xlsx: %w", err)
	}

	xf, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("open xlsx: %w", err)
	}
	defer xf.Close()

	var sb strings.Builder
	for _, sheet := range xf.GetSheetList() {
		rows, err := xf.GetRows(sheet)
		if err != nil {
			continue
		}
		for _, row := range rows {
			sb.WriteString(strings.Join(row, "\t"))
			sb.WriteString("\n")
		}
	}
	return strings.TrimSpace(sb.String()), nil
}
