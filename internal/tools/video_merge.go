package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// VideoMergeTool merges video and audio files using FFmpeg.
// Requires ffmpeg on PATH.
type VideoMergeTool struct {
	OutputDir string
}

func (v *VideoMergeTool) Name() string { return "video_merge" }

func (v *VideoMergeTool) Description() string {
	return "Merge video and audio files using FFmpeg. Modes: 'mux_audio' (add audio track to video), 'concat' (sequential join). Requires ffmpeg on PATH."
}

func (v *VideoMergeTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"inputs": map[string]any{
				"type":        "array",
				"description": "Array of input file paths. For mux_audio: [video_path, audio_path]. For concat: all video paths in order.",
				"items":       map[string]any{"type": "string"},
			},
			"mode": map[string]any{
				"type":        "string",
				"enum":        []any{"mux_audio", "concat"},
				"description": "Merge mode: 'mux_audio' adds an audio track to video; 'concat' joins clips sequentially",
			},
			"output_format": map[string]any{
				"type":        "string",
				"description": "Output container format (default: mp4)",
			},
		},
		"required": []any{"inputs"},
	}
}

func (v *VideoMergeTool) Execute(ctx context.Context, input any) (any, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("video_merge: invalid input")
	}

	rawInputs, ok := args["inputs"]
	if !ok {
		return nil, fmt.Errorf("video_merge: 'inputs' is required")
	}
	inputsArr, ok := rawInputs.([]any)
	if !ok {
		return nil, fmt.Errorf("video_merge: 'inputs' must be an array")
	}
	if len(inputsArr) == 0 {
		return nil, fmt.Errorf("video_merge: 'inputs' must have at least one element")
	}

	var inputs []string
	for _, p := range inputsArr {
		s, ok := p.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("video_merge: all inputs must be non-empty strings")
		}
		inputs = append(inputs, s)
	}

	mode, _ := args["mode"].(string)
	if mode == "" {
		mode = "mux_audio"
	}
	format, _ := args["output_format"].(string)
	if format == "" {
		format = "mp4"
	}

	outDir := v.OutputDir
	if outDir == "" {
		outDir = os.TempDir()
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("video_merge: create output dir: %w", err)
	}
	outPath := filepath.Join(outDir, uuid.New().String()+"."+format)

	var ffmpegArgs []string
	switch mode {
	case "mux_audio":
		if len(inputs) < 2 {
			return nil, fmt.Errorf("video_merge: mux_audio requires [video, audio]")
		}
		ffmpegArgs = []string{
			"-i", inputs[0],
			"-i", inputs[1],
			"-c:v", "copy",
			"-c:a", "aac",
			"-map", "0:v:0",
			"-map", "1:a:0",
			"-shortest",
			outPath,
		}
	case "concat":
		listPath := filepath.Join(outDir, uuid.New().String()+".txt")
		var sb strings.Builder
		for _, p := range inputs {
			fmt.Fprintf(&sb, "file '%s'\n", p)
		}
		if err := os.WriteFile(listPath, []byte(sb.String()), 0644); err != nil {
			return nil, fmt.Errorf("video_merge: write concat list: %w", err)
		}
		defer os.Remove(listPath)
		ffmpegArgs = []string{
			"-f", "concat",
			"-safe", "0",
			"-i", listPath,
			"-c", "copy",
			outPath,
		}
	default:
		return nil, fmt.Errorf("video_merge: unknown mode %q (supported: mux_audio, concat)", mode)
	}

	execCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "ffmpeg", append([]string{"-y"}, ffmpegArgs...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("video_merge: ffmpeg failed: %w\n%s", err, string(out))
	}

	return outPath, nil
}
