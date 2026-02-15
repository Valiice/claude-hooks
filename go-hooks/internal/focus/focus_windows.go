package focus

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                    = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow   = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcID = user32.NewProc("GetWindowThreadProcessId")
)

// TerminalIsFocused returns true if the foreground window belongs to an
// ancestor process of the current process (i.e. our terminal is focused).
// Returns false on any error (fail-open: show the notification).
func TerminalIsFocused() bool {
	fgPID, err := getForegroundPID()
	if err != nil || fgPID == 0 {
		return false
	}

	parentOf, err := buildProcessMap()
	if err != nil {
		return false
	}

	ancestors := getAncestors(uint32(os.Getpid()), parentOf)
	for _, a := range ancestors {
		if a == fgPID {
			return true
		}
	}
	return false
}

func getForegroundPID() (uint32, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0, windows.ERROR_INVALID_HANDLE
	}
	var pid uint32
	_, _, err := procGetWindowThreadProcID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return 0, err
	}
	return pid, nil
}

func buildProcessMap() (map[uint32]uint32, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)

	parentOf := make(map[uint32]uint32)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	err = windows.Process32First(snap, &entry)
	if err != nil {
		return nil, err
	}
	for {
		parentOf[entry.ProcessID] = entry.ParentProcessID
		err = windows.Process32Next(snap, &entry)
		if err != nil {
			break
		}
	}
	return parentOf, nil
}

// getAncestors walks up the process tree from startPID, returning all ancestor PIDs.
// Stops at depth 20 or on cycle/missing parent.
func getAncestors(startPID uint32, parentOf map[uint32]uint32) []uint32 {
	var ancestors []uint32
	seen := make(map[uint32]bool)
	seen[startPID] = true
	current := startPID

	for i := 0; i < 20; i++ {
		parent, ok := parentOf[current]
		if !ok || parent == 0 || seen[parent] {
			break
		}
		ancestors = append(ancestors, parent)
		seen[parent] = true
		current = parent
	}
	return ancestors
}
