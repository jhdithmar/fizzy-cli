package commands

import (
	"strings"
	"testing"

	"github.com/basecamp/cli/output"
)

func TestCommandsStyledOutputRendersHumanCatalog(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestFormat(output.FormatStyled)
	defer resetTest()

	if err := commandsCmd.RunE(commandsCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := TestOutput()
	if !strings.Contains(raw, "Name") {
		t.Fatalf("expected styled catalog header, got:\n%s", raw)
	}
	if !strings.Contains(raw, "fizzy auth") {
		t.Fatalf("expected styled catalog to include commands, got:\n%s", raw)
	}
}

func TestCommandsJSONOutputReturnsStructuredCatalog(t *testing.T) {
	mock := NewMockClient()
	result := SetTestModeWithSDK(mock)
	defer resetTest()

	if err := commandsCmd.RunE(commandsCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Response == nil || !result.Response.OK {
		t.Fatalf("expected OK JSON response, got %#v", result.Response)
	}

	items, ok := result.Response.Data.([]any)
	if !ok {
		t.Fatalf("expected command catalog slice, got %#v", result.Response.Data)
	}
	if len(items) == 0 {
		t.Fatal("expected command catalog entries")
	}

	found := false
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if entry["name"] == "fizzy commands" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected command catalog to include fizzy commands, got %#v", items)
	}
}
