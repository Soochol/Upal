package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// RemotionRenderTool renders a Remotion React composition to video.
// Requires Node.js and @remotion/renderer installed in the working directory
// or globally. The composition code is written to a temp directory and
// rendered via `npx remotion render`.
type RemotionRenderTool struct {
	OutputDir    string
	RemotionRoot string // path to Remotion project root (default: ./remotion)
}

func (r *RemotionRenderTool) Name() string { return "remotion_render" }

func (r *RemotionRenderTool) Description() string {
	return "Render a Remotion React composition to an MP4 video file. Requires Node.js and @remotion/renderer on the host. composition_code is the full React/Remotion source. audio_path is the TTS audio file to use."
}

func (r *RemotionRenderTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"composition_code": map[string]any{
				"type":        "string",
				"description": "Full Remotion React composition source code (TypeScript/JavaScript)",
			},
			"audio_path": map[string]any{
				"type":        "string",
				"description": "Path to the TTS audio file to include in the video",
			},
			"duration_sec": map[string]any{
				"type":        "number",
				"description": "Video duration in seconds (default: 60)",
			},
			"fps": map[string]any{
				"type":        "number",
				"description": "Frames per second (default: 30)",
			},
			"width": map[string]any{
				"type":        "number",
				"description": "Video width in pixels (default: 1920)",
			},
			"height": map[string]any{
				"type":        "number",
				"description": "Video height in pixels (default: 1080)",
			},
		},
		"required": []any{"composition_code"},
	}
}

func (r *RemotionRenderTool) Execute(ctx context.Context, input any) (any, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("remotion_render: invalid input")
	}

	code, _ := args["composition_code"].(string)
	if code == "" {
		return nil, fmt.Errorf("remotion_render: composition_code is required")
	}

	audioPath, _ := args["audio_path"].(string)
	durationSec, _ := args["duration_sec"].(float64)
	if durationSec <= 0 {
		durationSec = 60
	}
	fps, _ := args["fps"].(float64)
	if fps <= 0 {
		fps = 30
	}
	width, _ := args["width"].(float64)
	if width <= 0 {
		width = 1920
	}
	height, _ := args["height"].(float64)
	if height <= 0 {
		height = 1080
	}

	tmpDir, err := os.MkdirTemp("", "upal-remotion-*")
	if err != nil {
		return nil, fmt.Errorf("remotion_render: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	codePath := filepath.Join(tmpDir, "Composition.tsx")
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("remotion_render: write composition code: %w", err)
	}

	outDir := r.OutputDir
	if outDir == "" {
		outDir = os.TempDir()
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("remotion_render: create output dir: %w", err)
	}
	outPath := filepath.Join(outDir, uuid.New().String()+".mp4")

	remotionRoot := r.RemotionRoot
	if remotionRoot == "" {
		remotionRoot = "./remotion"
	}

	propsJSON := fmt.Sprintf(`{"audioPath":%q,"durationInSeconds":%g}`, audioPath, durationSec)

	npxArgs := []string{
		"remotion", "render",
		remotionRoot,
		"Composition",
		outPath,
		fmt.Sprintf("--width=%d", int(width)),
		fmt.Sprintf("--height=%d", int(height)),
		fmt.Sprintf("--fps=%d", int(fps)),
		fmt.Sprintf("--frames=0-%d", int(durationSec*fps)-1),
		"--props=" + propsJSON,
		"--overwrite",
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "npx", npxArgs...)
	cmd.Env = append(os.Environ(), "COMPOSITION_FILE="+codePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("remotion_render: render failed: %w\n%s", err, string(out))
	}

	return outPath, nil
}
