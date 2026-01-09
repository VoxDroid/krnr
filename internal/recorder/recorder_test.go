package recorder

import (
	"strings"
	"testing"
)

func TestRecordCommands_IgnoresBlankAndComments(t *testing.T) {
	input := "# comment line\necho one\n\n# another comment\necho two  \n"
	cmds, err := RecordCommands(strings.NewReader(input))
	if err != nil {
		t.Fatalf("RecordCommands: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0] != "echo one" || cmds[1] != "echo two" {
		t.Fatalf("unexpected commands: %+v", cmds)
	}
}

func TestRecordCommands_StopsOnCtrlZAlone(t *testing.T) {
	// Ctrl+Z alone should be treated as EOF
	cmds, err := RecordCommands(strings.NewReader("\x1A"))
	if err != nil {
		t.Fatalf("RecordCommands ctrl+Z: %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(cmds))
	}
}

func TestRecordCommands_StopsOnCtrlZMidInput(t *testing.T) {
	// Data after Ctrl+Z should be ignored
	input := "echo before\x1Aecho after\n"
	cmds, err := RecordCommands(strings.NewReader(input))
	if err != nil {
		t.Fatalf("RecordCommands ctrl+Z mid: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0] != "echo before" {
		t.Fatalf("unexpected command: %+v", cmds)
	}
}

func TestRecordCommands_StopsOnCaretZAlone(t *testing.T) {
	// '^Z' on its own (as typed in some consoles) should be treated as EOF
	cmds, err := RecordCommands(strings.NewReader("^Z\n"))
	if err != nil {
		t.Fatalf("RecordCommands ^Z: %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(cmds))
	}
}

func TestRecordCommands_StopsOnCaretZMidInput(t *testing.T) {
	// Data after '^Z' within a line should be ignored
	input := "echo before^Zecho after\n"
	cmds, err := RecordCommands(strings.NewReader(input))
	if err != nil {
		t.Fatalf("RecordCommands ^Z mid: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0] != "echo before" {
		t.Fatalf("unexpected command: %+v", cmds)
	}
}

func TestRecordCommands_StopsOnSentinelAlone(t *testing.T) {
	cmds, err := RecordCommands(strings.NewReader(":end\n"))
	if err != nil {
		t.Fatalf("RecordCommands sentinel :end: %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(cmds))
	}
}

func TestRecordCommands_StopsOnSentinelAliases(t *testing.T) {
	for _, s := range []string{":save\n", ":quit\n", "  :end  \n"} {
		cmds, err := RecordCommands(strings.NewReader(s))
		if err != nil {
			t.Fatalf("RecordCommands sentinel alias %s: %v", s, err)
		}
		if len(cmds) != 0 {
			t.Fatalf("expected 0 commands for %s, got %d", s, len(cmds))
		}
	}
}

func TestRecordCommands_SentinelStopsMidStream(t *testing.T) {
	input := "echo one\n:end\necho two\n"
	cmds, err := RecordCommands(strings.NewReader(input))
	if err != nil {
		t.Fatalf("RecordCommands sentinel mid: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0] != "echo one" {
		t.Fatalf("unexpected commands: %+v", cmds)
	}
}

func TestRecordCommands_SentinelNotMatchedWithExtraText(t *testing.T) {
	input := "echo :end something\n"
	cmds, err := RecordCommands(strings.NewReader(input))
	if err != nil {
		t.Fatalf("RecordCommands sentinel extra: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0] != "echo :end something" {
		t.Fatalf("unexpected command: %+v", cmds)
	}
}
