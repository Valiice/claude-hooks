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

	"github.com/valentinclaes/claude-hooks/internal/gitctx"
	"github.com/valentinclaes/claude-hooks/internal/gitsync"
	"github.com/valentinclaes/claude-hooks/internal/hookdata"
	"github.com/valentinclaes/claude-hooks/internal/obsidian"
	"github.com/valentinclaes/claude-hooks/internal/session"
	"github.com/valentinclaes/claude-hooks/internal/transcript"
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
		session.Write(input.SessionID, filePath, promptNum, sd.Branch, sd.StartHash, sd.Cwd)
	} else {
		// New session
		promptNum = 1
		projectDir := filepath.Join(vaultDir, project)
		os.MkdirAll(projectDir, 0755)

		timeShort := now.Format("1504")
		fileName := fmt.Sprintf("%s_%s.md", date, timeShort)
		filePath = filepath.Join(projectDir, fileName)

		// Handle collision
		counter := 2
		for {
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				break
			}
			fileName = fmt.Sprintf("%s_%s_%d.md", date, timeShort, counter)
			filePath = filepath.Join(projectDir, fileName)
			counter++
		}

		// Capture git context
		gc := gitctx.Capture(input.Cwd)
		session.Write(input.SessionID, filePath, 1, gc.Branch, gc.Hash, input.Cwd)

		// Check for parent session
		resumedFrom := obsidian.FindParentSession(input.SessionID, claudeProjects, vaultDir)

		startTime := now.Format("15:04")
		frontmatter := obsidian.BuildFrontmatter(obsidian.FrontmatterData{
			Date:        date,
			SessionID:   input.SessionID,
			Project:     project,
			StartTime:   startTime,
			ResumedFrom: resumedFrom,
			Branch:      gc.Branch,
		})
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

	// Read transcript and find last assistant text + planContent
	responseText, planText := readTranscript(input.TranscriptPath)

	// Parse transcript for stats
	stats, _ := transcript.ParseTranscript(input.TranscriptPath)

	now := time.Now()
	timeStr := now.Format("15:04:05")
	var output strings.Builder

	// Add stats summary line if we have stats
	if stats != nil && stats.TotalToolCalls() > 0 {
		estCost := ""
		if stats.EstimatedCost > 0 {
			estCost = fmt.Sprintf("$%.2f", stats.EstimatedCost)
		}
		statsLine := obsidian.FormatStatsLine(stats.ToolCounts, stats.TokensIn, stats.TokensOut, estCost)
		if statsLine != "" {
			output.WriteString("\n" + statsLine)
		}
	}

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

	// Detect git commits since last check
	var commitLines []string
	if sd.StartHash != "" && sd.Cwd != "" {
		commits := gitctx.CommitsSince(sd.Cwd, sd.StartHash)
		for _, c := range commits {
			commitLines = append(commitLines, c.Hash+" "+c.Message)
		}
		if len(commits) > 0 {
			// Advance start hash to current HEAD so next response only sees new commits
			newHead := gitctx.Capture(sd.Cwd)
			if newHead.Hash != "" {
				session.Write(input.SessionID, sd.FilePath, sd.PromptNum, sd.Branch, newHead.Hash, sd.Cwd)
			}
		}
	}
	if len(commitLines) > 0 {
		output.WriteString(obsidian.FormatCommitsEntry(now.Format("15:04"), commitLines))
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

	// Update frontmatter with stats
	if stats != nil && stats.TotalToolCalls() > 0 {
		estCost := ""
		if stats.EstimatedCost > 0 {
			estCost = fmt.Sprintf("$%.2f", stats.EstimatedCost)
		}
		content, err := os.ReadFile(filePath)
		if err == nil {
			updated := obsidian.UpdateFrontmatterStats(
				string(content),
				stats.ToolCounts, stats.TokensIn, stats.TokensOut,
				stats.CacheRead, stats.CacheCreation,
				estCost, stats.FilesTouched, commitLines, sd.Branch, stats.Model,
			)
			os.WriteFile(filePath, []byte(updated), 0644)
		}
	}

	// Rebuild daily index
	vaultDir := obsidian.VaultDir()
	if vaultDir != "" {
		date := now.Format("2006-01-02")
		obsidian.RebuildDailyIndex(vaultDir, date)
		obsidian.RebuildWeeklyStatsIfStale(vaultDir, now)
		obsidian.RebuildMonthlyStatsIfStale(vaultDir, now)

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

	// Walk backwards, up to 50 lines
	maxLook := 50
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
