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
