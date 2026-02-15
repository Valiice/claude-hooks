package focus

import (
	"os"
	"testing"
)

func TestGetAncestors_SimpleChain(t *testing.T) {
	// 1 -> 2 -> 3 -> 4
	parentOf := map[uint32]uint32{1: 2, 2: 3, 3: 4}
	ancestors := getAncestors(1, parentOf)

	want := []uint32{2, 3, 4}
	if len(ancestors) != len(want) {
		t.Fatalf("got %v, want %v", ancestors, want)
	}
	for i, v := range want {
		if ancestors[i] != v {
			t.Fatalf("got %v, want %v", ancestors, want)
		}
	}
}

func TestGetAncestors_CycleDetection(t *testing.T) {
	// 1 -> 2 -> 3 -> 2 (cycle)
	parentOf := map[uint32]uint32{1: 2, 2: 3, 3: 2}
	ancestors := getAncestors(1, parentOf)

	want := []uint32{2, 3}
	if len(ancestors) != len(want) {
		t.Fatalf("got %v, want %v", ancestors, want)
	}
	for i, v := range want {
		if ancestors[i] != v {
			t.Fatalf("got %v, want %v", ancestors, want)
		}
	}
}

func TestGetAncestors_MissingParent(t *testing.T) {
	// 1 -> 2, but 2 has no parent in map
	parentOf := map[uint32]uint32{1: 2}
	ancestors := getAncestors(1, parentOf)

	if len(ancestors) != 1 || ancestors[0] != 2 {
		t.Fatalf("got %v, want [2]", ancestors)
	}
}

func TestGetAncestors_SelfParent(t *testing.T) {
	// 1 -> 1 (self-parent, should stop immediately via seen check)
	parentOf := map[uint32]uint32{1: 1}
	ancestors := getAncestors(1, parentOf)

	if len(ancestors) != 0 {
		t.Fatalf("got %v, want []", ancestors)
	}
}

func TestGetAncestors_EmptyMap(t *testing.T) {
	parentOf := map[uint32]uint32{}
	ancestors := getAncestors(1, parentOf)

	if len(ancestors) != 0 {
		t.Fatalf("got %v, want []", ancestors)
	}
}

func TestBuildProcessMap_Succeeds(t *testing.T) {
	parentOf, err := buildProcessMap()
	if err != nil {
		t.Fatalf("buildProcessMap failed: %v", err)
	}

	myPID := uint32(os.Getpid())
	if _, ok := parentOf[myPID]; !ok {
		t.Fatalf("our PID %d not found in process map", myPID)
	}
}

func TestGetForegroundPID_Succeeds(t *testing.T) {
	pid, err := getForegroundPID()
	if err != nil || pid == 0 {
		t.Skip("no foreground window (headless environment)")
	}
}

func TestTerminalIsFocused_DoesNotPanic(t *testing.T) {
	// Smoke test — just make sure it doesn't panic.
	_ = TerminalIsFocused()
}
