package main

import (
	"os"

	"github.com/gen2brain/beeep"
	"github.com/valentinclaes/claude-hooks/internal/config"
	"github.com/valentinclaes/claude-hooks/internal/focus"
)

func main() {
	defer func() {
		recover() // Never block Claude - swallow all panics
	}()

	cfg := config.Load()
	if cfg.SkipWhenFocused && focus.TerminalIsFocused() {
		return
	}

	beeep.AppName = "Claude Code"

	title := "Claude"
	message := "Task completed!"

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--title":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			}
		case "--message":
			if i+1 < len(args) {
				message = args[i+1]
				i++
			}
		}
	}

	beeep.Alert(title, message, "")
}
