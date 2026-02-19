package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/valentinclaes/claude-hooks/internal/gitsync"
	"github.com/valentinclaes/claude-hooks/internal/hookdata"
	"github.com/valentinclaes/claude-hooks/internal/obsidian"
	"github.com/valentinclaes/claude-hooks/internal/session"
)

var startTimeRe = regexp.MustCompile(`(?m)^start_time:\s*(\d{2}:\d{2})`)
var durationLineRe = regexp.MustCompile(`(?m)^duration:.*$`)
var startTimeLineRe = regexp.MustCompile(`(?m)^(start_time:\s*.*)$`)

func main() {
	defer func() {
		recover() // Never block Claude - swallow all panics
	}()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: claude-obsidian <log-prompt|log-response>")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "log-prompt":
		runLogPrompt()
	case "log-response":
		runLogResponse()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
	}
}

func runLogPrompt() {
	var input hookdata.PromptInput
	if err := hookdata.ReadStdin(&input); err != nil {
		return
	}
	if input.Prompt == "" {
		return
	}

	prompt := obsidian.StripSystemTags(input.Prompt)
	if prompt == "" {
		return
	}
	prompt = obsidian.Truncate(prompt, 5000)

	vaultDir := obsidian.VaultDir()
	if vaultDir == "" {
		return
	}

	claudeProjects := filepath.Join(os.Getenv("USERPROFILE"), ".claude", "projects")
	now := time.Now()
	date := now.Format("2006-01-02")
	timeStr := now.Format("15:04:05")
	project := obsidian.SanitizeProject(filepath.Base(input.Cwd))

	// Ensure vault dir exists
	os.MkdirAll(vaultDir, 0755)

	// Clean up stale session files
	session.CleanupStale()

	// Check for existing session mapping
	sd, _ := session.Read(input.SessionID)

	var filePath string
	var promptNum int

	if sd != nil {
		filePath = sd.FilePath
		promptNum = sd.PromptNum + 1
		session.Write(input.SessionID, filePath, promptNum)
	} else {
		// New session
		promptNum = 1
		projectDir := filepath.Join(vaultDir, project)
		os.MkdirAll(projectDir, 0755)

		timeShort := now.Format("1504")
		titleSlug := obsidian.GenerateTitleSlug(prompt)
		var fileName string
		if titleSlug != "" {
			fileName = fmt.Sprintf("%s_%s_%s.md", date, timeShort, titleSlug)
		} else {
			fileName = fmt.Sprintf("%s_%s.md", date, timeShort)
		}
		filePath = filepath.Join(projectDir, fileName)

		// Handle collision
		counter := 2
		for {
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				break
			}
			if titleSlug != "" {
				fileName = fmt.Sprintf("%s_%s_%s_%d.md", date, timeShort, titleSlug, counter)
			} else {
				fileName = fmt.Sprintf("%s_%s_%d.md", date, timeShort, counter)
			}
			filePath = filepath.Join(projectDir, fileName)
			counter++
		}

		session.Write(input.SessionID, filePath, 1)

		// Check for parent session
		resumedFrom := obsidian.FindParentSession(input.SessionID, claudeProjects, vaultDir)

		startTime := now.Format("15:04")
		frontmatter := obsidian.BuildFrontmatter(date, input.SessionID, project, startTime, resumedFrom)
		os.WriteFile(filePath, []byte(frontmatter), 0644)
	}

	// Append prompt entry
	entry := obsidian.FormatPromptEntry(promptNum, timeStr, input.Cwd, prompt)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(entry)
}

func runLogResponse() {
	var input hookdata.StopInput
	if err := hookdata.ReadStdin(&input); err != nil {
		return
	}
	if input.TranscriptPath == "" {
		return
	}
	if _, err := os.Stat(input.TranscriptPath); err != nil {
		return
	}

	sd, _ := session.Read(input.SessionID)
	if sd == nil {
		return
	}
	filePath := sd.FilePath
	if _, err := os.Stat(filePath); err != nil {
		return
	}

	// Read transcript and find last assistant text + planContent.
	// Retry up to 3 times with 500ms delay to handle the race where the Stop
	// hook fires before the transcript is fully flushed to disk.
	var responseText, planText string
	for attempt := 0; attempt < 3; attempt++ {
		responseText, planText = readTranscript(input.TranscriptPath)
		if responseText != "" || planText != "" {
			break
		}
		if attempt < 2 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	now := time.Now()
	timeStr := now.Format("15:04:05")
	var output strings.Builder

	// Log plan if found
	if planText != "" {
		planText = obsidian.TruncateSimple(planText, 5000)
		output.WriteString(obsidian.FormatPlanEntry(timeStr, planText))
	}

	// Log response
	if responseText != "" {
		responseText = obsidian.Truncate(responseText, 3000)
		output.WriteString(obsidian.FormatResponseEntry(timeStr, responseText))
	}

	if output.Len() > 0 {
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		f.WriteString(output.String())
		f.Close()
	}

	// Update duration in frontmatter
	updateDuration(filePath, now)

	// Insert session summary block
	topics := readUserTopics(input.TranscriptPath)
	if len(topics) > 0 {
		insertSummaryBlock(filePath, obsidian.FormatSummaryBlock(topics))
	}

	// Rebuild daily index
	vaultDir := obsidian.VaultDir()
	if vaultDir != "" {
		date := now.Format("2006-01-02")
		obsidian.RebuildDailyIndex(vaultDir, date)

		// Git sync (if enabled via config.json)
		gitsync.SyncIfEnabled(vaultDir)
	}
}

// transcriptMessage represents the minimal structure we need from transcript JSONL.
type transcriptMessage struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
	PlanContent string `json:"planContent"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func readTranscript(path string) (responseText, planText string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	// Read all lines into memory (we need to walk backwards)
	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Walk backwards, up to 200 lines
	maxLook := 200
	if maxLook > len(lines) {
		maxLook = len(lines)
	}

	for i := len(lines) - 1; i >= len(lines)-maxLook; i-- {
		if i < 0 {
			break
		}
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var msg transcriptMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		// Check for plan content
		if planText == "" && msg.PlanContent != "" {
			planText = msg.PlanContent
		}

		// Check for assistant text response
		if responseText == "" && msg.Type == "assistant" && msg.Message.Role == "assistant" {
			var blocks []contentBlock
			if err := json.Unmarshal(msg.Message.Content, &blocks); err != nil {
				continue
			}
			var texts []string
			for _, b := range blocks {
				if b.Type == "text" && b.Text != "" {
					texts = append(texts, b.Text)
				}
			}
			if len(texts) > 0 {
				responseText = strings.Join(texts, "\n\n")
				break
			}
		}
	}
	return
}

// readUserTopics scans the transcript JSONL forward and returns up to 10 first-line
// summaries of user turns (system tags stripped, truncated at 100 chars).
func readUserTopics(transcriptPath string) []string {
	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var topics []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		if len(topics) >= 10 {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg transcriptMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Message.Role != "user" {
			continue
		}
		var text string
		var blocks []contentBlock
		if err := json.Unmarshal(msg.Message.Content, &blocks); err == nil {
			for _, b := range blocks {
				if b.Type == "text" && b.Text != "" {
					text = b.Text
					break
				}
			}
		} else {
			var rawStr string
			if err := json.Unmarshal(msg.Message.Content, &rawStr); err == nil {
				text = rawStr
			}
		}
		if text == "" {
			continue
		}
		text = obsidian.StripSystemTags(text)
		if text == "" {
			continue
		}
		if idx := strings.IndexByte(text, '\n'); idx >= 0 {
			text = text[:idx]
		}
		text = truncateTopic(strings.TrimSpace(text), 100)
		if text == "" {
			continue
		}
		topics = append(topics, text)
	}
	return topics
}

func truncateTopic(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	truncated := s[:maxLen]
	if idx := strings.LastIndexByte(truncated, ' '); idx > 0 {
		truncated = truncated[:idx]
	}
	return truncated + "..."
}

// insertSummaryBlock inserts the summary block into the session file immediately
// before the first "\n---\n" that follows the "# Claude Session - " heading.
func insertSummaryBlock(filePath, summaryBlock string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	contentStr := string(content)

	// Idempotency guard: don't insert if block already exists
	if strings.Contains(contentStr, "**Topics covered:**") {
		return
	}

	headingMarker := "\n# Claude Session - "
	headingIdx := strings.Index(contentStr, headingMarker)
	if headingIdx < 0 {
		return
	}

	searchFrom := headingIdx + len(headingMarker)
	sepMarker := "\n---\n"
	sepIdx := strings.Index(contentStr[searchFrom:], sepMarker)
	if sepIdx < 0 {
		return
	}
	insertAt := searchFrom + sepIdx

	newContent := contentStr[:insertAt] + "\n" + summaryBlock + contentStr[insertAt:]
	os.WriteFile(filePath, []byte(newContent), 0644)
}

func updateDuration(filePath string, now time.Time) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	contentStr := string(content)

	m := startTimeRe.FindStringSubmatch(contentStr)
	if len(m) < 2 {
		return
	}

	startStr := m[1]
	today := now.Format("2006-01-02")
	startDt, err := time.ParseInLocation("2006-01-02 15:04", today+" "+startStr, time.Local)
	if err != nil {
		return
	}

	dur := now.Sub(startDt)
	totalMin := int(math.Floor(dur.Minutes()))
	if totalMin < 1 {
		totalMin = 1
	}
	durStr := fmt.Sprintf("%dmin", totalMin)

	if durationLineRe.MatchString(contentStr) {
		contentStr = durationLineRe.ReplaceAllString(contentStr, "duration: "+durStr)
	} else {
		contentStr = startTimeLineRe.ReplaceAllString(contentStr, "${1}\nduration: "+durStr)
	}

	os.WriteFile(filePath, []byte(contentStr), 0644)
}
