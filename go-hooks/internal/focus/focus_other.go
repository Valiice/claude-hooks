//go:build !windows

package focus

// TerminalIsFocused always returns false on non-Windows platforms,
// meaning notifications are always shown.
func TerminalIsFocused() bool {
	return false
}
