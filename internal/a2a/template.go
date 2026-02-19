package a2a

import (
	"encoding/json"
	"regexp"
	"strings"
)

var templatePattern = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

// ResolveTemplate resolves {{node_id}} references from artifact-based state.
//
// Supported patterns:
//   - {{node_id}}      → first text part of the node's first artifact
//   - {{node_id.data}} → JSON-serialized first data part
func ResolveTemplate(template string, artifacts map[string][]Artifact) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.Trim(match, "{}")
		parts := strings.SplitN(key, ".", 2)
		nodeID := parts[0]

		arts, ok := artifacts[nodeID]
		if !ok || len(arts) == 0 {
			return match
		}
		art := arts[0]

		if len(parts) == 2 && parts[1] == "data" {
			data := art.FirstData()
			if data == nil {
				return match
			}
			return string(data)
		}

		// Default: first text part
		text := art.FirstText()
		if text == "" {
			// Fallback: serialize the first data part
			data := art.FirstData()
			if data != nil {
				return string(data)
			}
			// Last resort: JSON of the whole artifact
			b, _ := json.Marshal(art)
			return string(b)
		}
		return text
	})
}
