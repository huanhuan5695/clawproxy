package server

import (
	"context"
	"testing"
)

func TestBuildOpenClawCommand(t *testing.T) {
	cmd := buildOpenClawCommand(context.Background(), "dev-123", "hello world")

	if cmd.Path != "openclaw" {
		t.Fatalf("expected command path %q, got %q", "openclaw", cmd.Path)
	}

	expectedArgs := []string{"openclaw", "agent", "--session-id", "dev-123", "--message", "hello world", "--json"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d: %#v", len(expectedArgs), len(cmd.Args), cmd.Args)
	}

	for i := range expectedArgs {
		if cmd.Args[i] != expectedArgs[i] {
			t.Fatalf("arg[%d]: expected %q, got %q", i, expectedArgs[i], cmd.Args[i])
		}
	}
}

func TestExtractJSONObject(t *testing.T) {
	out := "normal logs... {\"answer\":\"ok\",\"code\":200} trailing"
	jsonPart, err := extractJSONObject(out)
	if err != nil {
		t.Fatalf("extract json object: %v", err)
	}

	if jsonPart != `{"answer":"ok","code":200}` {
		t.Fatalf("unexpected json part: %s", jsonPart)
	}
}
