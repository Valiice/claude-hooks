package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"
	"strings"
)

// SessionStats holds aggregated statistics from a transcript.
type SessionStats struct {
	ToolCounts    map[string]int
	FilesTouched  []string
	TokensIn      int
	TokensOut     int
	CacheRead     int
	CacheCreation int
	EstimatedCost float64
	Model         string // "opus", "sonnet", "haiku", or "" if unknown
}

// modelRates holds per-million-token pricing for a model family.
type modelRates struct {
	Input  float64
	Output float64
}

// ratesByModel maps normalized model family to pricing.
var ratesByModel = map[string]modelRates{
	"opus":   {5.0, 25.0},
	"sonnet": {3.0, 15.0},
	"haiku":  {1.0, 5.0},
}

// resolveRates returns pricing for a normalized model name, defaulting to Sonnet.
func resolveRates(model string) modelRates {
	if r, ok := ratesByModel[model]; ok {
		return r
	}
	return ratesByModel["sonnet"]
}

// normalizeModel extracts the model family from a full model ID.
// e.g. "claude-opus-4-6" -> "opus", "claude-sonnet-4-5-20250929" -> "sonnet".
func normalizeModel(raw string) string {
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "opus"):
		return "opus"
	case strings.Contains(lower, "sonnet"):
		return "sonnet"
	case strings.Contains(lower, "haiku"):
		return "haiku"
	}
	return ""
}

// ParseTranscript reads a JSONL transcript file and returns aggregated stats.
func ParseTranscript(path string) (*SessionStats, error) {
	defer func() { recover() }()

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stats := &SessionStats{
		ToolCounts: make(map[string]int),
	}
	fileSet := make(map[string]bool)
	detectedModel := ""

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg transcriptMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		// Skip sidechain messages
		if msg.IsSidechain {
			continue
		}

		// Process assistant messages for tokens and tool usage
		if msg.Type == "assistant" && msg.Message.Role == "assistant" {
			// Capture model from first assistant message
			if detectedModel == "" && msg.Message.Model != "" {
				detectedModel = normalizeModel(msg.Message.Model)
			}

			// Sum tokens
			stats.TokensIn += msg.Message.Usage.InputTokens
			stats.TokensOut += msg.Message.Usage.OutputTokens
			stats.CacheRead += msg.Message.Usage.CacheReadInputTokens
			stats.CacheCreation += msg.Message.Usage.CacheCreationInputTokens

			// Count tools and extract file paths
			var blocks []contentBlock
			if err := json.Unmarshal(msg.Message.Content, &blocks); err != nil {
				continue
			}
			for _, b := range blocks {
				if b.Type == "tool_use" && b.Name != "" {
					stats.ToolCounts[b.Name]++
					extractFilePaths(b.Input, b.Name, fileSet)
				}
			}
		}
	}

	stats.Model = detectedModel

	// Calculate cost with model-aware rates and cache token accounting.
	// API fields are separate: input_tokens, cache_read, cache_creation, output_tokens.
	rates := resolveRates(detectedModel)
	stats.EstimatedCost = float64(stats.TokensIn)/1e6*rates.Input +
		float64(stats.CacheRead)/1e6*(rates.Input*0.1) +
		float64(stats.CacheCreation)/1e6*(rates.Input*1.25) +
		float64(stats.TokensOut)/1e6*rates.Output

	// Convert file set to sorted slice
	for f := range fileSet {
		stats.FilesTouched = append(stats.FilesTouched, f)
	}
	sort.Strings(stats.FilesTouched)

	return stats, nil
}

// TotalToolCalls returns the sum of all tool call counts.
func (s *SessionStats) TotalToolCalls() int {
	total := 0
	for _, c := range s.ToolCounts {
		total += c
	}
	return total
}

// transcriptMessage represents a line in the transcript JSONL.
type transcriptMessage struct {
	Type        string `json:"type"`
	IsSidechain bool   `json:"isSidechain"`
	Message     struct {
		Model   string          `json:"model"`
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
		Usage   struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// contentBlock represents a content block in a message.
type contentBlock struct {
	Type  string          `json:"type"`
	Name  string          `json:"name"`
	Text  string          `json:"text"`
	Input json.RawMessage `json:"input"`
}

// extractFilePaths extracts file paths from tool input based on tool name.
func extractFilePaths(input json.RawMessage, toolName string, fileSet map[string]bool) {
	if len(input) == 0 {
		return
	}

	var params map[string]json.RawMessage
	if err := json.Unmarshal(input, &params); err != nil {
		return
	}

	// Tools that use "file_path"
	switch toolName {
	case "Read", "Edit", "Write":
		if raw, ok := params["file_path"]; ok {
			var fp string
			if json.Unmarshal(raw, &fp) == nil && fp != "" {
				fileSet[normalizePath(fp)] = true
			}
		}
	case "Grep", "Glob":
		if raw, ok := params["path"]; ok {
			var fp string
			if json.Unmarshal(raw, &fp) == nil && fp != "" {
				fileSet[normalizePath(fp)] = true
			}
		}
	}
}

// normalizePath converts backslashes to forward slashes for consistency.
func normalizePath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
