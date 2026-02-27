package transcript

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func writeTempJSONL(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		f.WriteString(line + "\n")
	}
	f.Close()
	return path
}

func TestParseTranscript_ToolCounts(t *testing.T) {
	lines := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/foo/bar.go"}},{"type":"tool_use","name":"Edit","input":{"file_path":"/foo/bar.go"}}],"usage":{"input_tokens":100,"output_tokens":50}}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/foo/baz.go"}},{"type":"tool_use","name":"Bash","input":{"command":"ls"}}],"usage":{"input_tokens":200,"output_tokens":100}}}`,
	}
	path := writeTempJSONL(t, lines)
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if stats.ToolCounts["Read"] != 2 {
		t.Errorf("Read count: got %d, want 2", stats.ToolCounts["Read"])
	}
	if stats.ToolCounts["Edit"] != 1 {
		t.Errorf("Edit count: got %d, want 1", stats.ToolCounts["Edit"])
	}
	if stats.ToolCounts["Bash"] != 1 {
		t.Errorf("Bash count: got %d, want 1", stats.ToolCounts["Bash"])
	}
	if stats.TotalToolCalls() != 4 {
		t.Errorf("TotalToolCalls: got %d, want 4", stats.TotalToolCalls())
	}
}

func TestParseTranscript_TokenSums(t *testing.T) {
	lines := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":1000,"output_tokens":500,"cache_read_input_tokens":200,"cache_creation_input_tokens":100}}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"world"}],"usage":{"input_tokens":2000,"output_tokens":1000,"cache_read_input_tokens":300,"cache_creation_input_tokens":50}}}`,
	}
	path := writeTempJSONL(t, lines)
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TokensIn != 3000 {
		t.Errorf("TokensIn: got %d, want 3000", stats.TokensIn)
	}
	if stats.TokensOut != 1500 {
		t.Errorf("TokensOut: got %d, want 1500", stats.TokensOut)
	}
	if stats.CacheRead != 500 {
		t.Errorf("CacheRead: got %d, want 500", stats.CacheRead)
	}
	if stats.CacheCreation != 150 {
		t.Errorf("CacheCreation: got %d, want 150", stats.CacheCreation)
	}
}

func TestParseTranscript_FileDedup(t *testing.T) {
	lines := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/foo/bar.go"}},{"type":"tool_use","name":"Edit","input":{"file_path":"/foo/bar.go"}}],"usage":{"input_tokens":100,"output_tokens":50}}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/foo/baz.go"}}],"usage":{"input_tokens":100,"output_tokens":50}}}`,
	}
	path := writeTempJSONL(t, lines)
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.FilesTouched) != 2 {
		t.Errorf("FilesTouched count: got %d, want 2", len(stats.FilesTouched))
	}
}

func TestParseTranscript_CostCalc_Sonnet(t *testing.T) {
	// No model field -> defaults to Sonnet rates ($3/$15)
	lines := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":1000000,"output_tokens":100000}}}`,
	}
	path := writeTempJSONL(t, lines)
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	// 1M input * $3/M + 100K output * $15/M = $3 + $1.5 = $4.5
	expected := 4.5
	if math.Abs(stats.EstimatedCost-expected) > 0.01 {
		t.Errorf("EstimatedCost: got %.2f, want %.2f", stats.EstimatedCost, expected)
	}
}

func TestParseTranscript_ModelDetection(t *testing.T) {
	lines := []string{
		`{"type":"assistant","message":{"model":"claude-opus-4-6","role":"assistant","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":100,"output_tokens":50}}}`,
		`{"type":"assistant","message":{"model":"claude-opus-4-6","role":"assistant","content":[{"type":"text","text":"bye"}],"usage":{"input_tokens":100,"output_tokens":50}}}`,
	}
	path := writeTempJSONL(t, lines)
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Model != "opus" {
		t.Errorf("Model: got %q, want %q", stats.Model, "opus")
	}
}

func TestParseTranscript_OpusPricing(t *testing.T) {
	lines := []string{
		`{"type":"assistant","message":{"model":"claude-opus-4-6","role":"assistant","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":1000000,"output_tokens":100000}}}`,
	}
	path := writeTempJSONL(t, lines)
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	// 1M input * $5/M + 100K output * $25/M = $5 + $2.5 = $7.5
	expected := 7.5
	if math.Abs(stats.EstimatedCost-expected) > 0.01 {
		t.Errorf("EstimatedCost (Opus): got %.2f, want %.2f", stats.EstimatedCost, expected)
	}
}

func TestParseTranscript_CacheCostReduction(t *testing.T) {
	// Realistic Opus session: tiny input_tokens, large cache_read and cache_creation
	lines := []string{
		`{"type":"assistant","message":{"model":"claude-opus-4-6","role":"assistant","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":3,"output_tokens":2,"cache_read_input_tokens":10773,"cache_creation_input_tokens":14829}}}`,
	}
	path := writeTempJSONL(t, lines)
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	// Opus rates: input=$5, output=$25
	// 3 input * $5/M = $0.000015
	// 10773 cache_read * ($5*0.1)/M = 10773 * $0.5/M = $0.005387
	// 14829 cache_creation * ($5*1.25)/M = 14829 * $6.25/M = $0.092681
	// 2 output * $25/M = $0.000050
	// Total ≈ $0.098133
	expected := 0.098
	if math.Abs(stats.EstimatedCost-expected) > 0.001 {
		t.Errorf("EstimatedCost (cache): got %.6f, want ~%.3f", stats.EstimatedCost, expected)
	}

	// Verify this is much less than naive calc (all tokens at full input rate)
	naiveCost := float64(3+10773+14829) / 1e6 * 5.0 // $0.128
	if stats.EstimatedCost >= naiveCost {
		t.Errorf("Cache cost (%f) should be less than naive cost (%f)", stats.EstimatedCost, naiveCost)
	}
}

func TestParseTranscript_EmptyFile(t *testing.T) {
	path := writeTempJSONL(t, []string{})
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalToolCalls() != 0 {
		t.Errorf("TotalToolCalls for empty: got %d, want 0", stats.TotalToolCalls())
	}
	if stats.TokensIn != 0 {
		t.Errorf("TokensIn for empty: got %d, want 0", stats.TokensIn)
	}
}

func TestParseTranscript_MalformedLines(t *testing.T) {
	lines := []string{
		"this is not json",
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":100,"output_tokens":50}}}`,
		"{malformed json too",
	}
	path := writeTempJSONL(t, lines)
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TokensIn != 100 {
		t.Errorf("TokensIn with malformed: got %d, want 100", stats.TokensIn)
	}
}

func TestParseTranscript_SkipSidechain(t *testing.T) {
	lines := []string{
		`{"type":"assistant","isSidechain":true,"message":{"role":"assistant","content":[{"type":"text","text":"sidechain"}],"usage":{"input_tokens":5000,"output_tokens":2000}}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"main"}],"usage":{"input_tokens":100,"output_tokens":50}}}`,
	}
	path := writeTempJSONL(t, lines)
	stats, err := ParseTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TokensIn != 100 {
		t.Errorf("TokensIn (skip sidechain): got %d, want 100", stats.TokensIn)
	}
}
