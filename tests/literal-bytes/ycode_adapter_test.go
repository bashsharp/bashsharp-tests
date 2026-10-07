package provider

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	api "github.com/qiangli/ycode/internal/api"
)

// This file is overlaid into ycode's provider package by the focused gate, so
// the test traverses the production adapter and its JSON tool Input field.
func TestSprint381LiteralBytes(t *testing.T) {
	root := os.Getenv("LITERAL_BYTES_ROOT")
	shell := os.Getenv("BASH_SHELL_BIN")
	if root == "" || shell == "" {
		t.Fatal("LITERAL_BYTES_ROOT and BASH_SHELL_BIN are required")
	}
	data, err := os.ReadFile(filepath.Join(root, "cases.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range strings.Split(string(data), "\n") {
		fields := strings.Split(row, "\t")
		if len(fields) != 2 || fields[0] == "" || strings.HasPrefix(fields[0], "#") {
			continue
		}
		name, wantHex := fields[0], strings.TrimSpace(fields[1])
		t.Run(name, func(t *testing.T) {
			reply, err := os.ReadFile(filepath.Join(root, "cases", name+".bsh"))
			if err != nil {
				t.Fatal(err)
			}
			arg, err := json.Marshal(map[string]string{"script": string(reply)})
			if err != nil {
				t.Fatal(err)
			}
			partial, _ := json.Marshal(map[string]string{"type": "input_json_delta", "partial_json": string(arg)})
			backend := &MockBackend{Events: []*api.StreamEvent{
				{Type: "content_block_start", Index: 1, ContentBlock: &api.ContentBlock{Type: api.ContentTypeToolUse, ID: "s381", Name: ToolName, Input: json.RawMessage(`{}`)}},
				{Type: "content_block_delta", Index: 1, Delta: partial},
				{Type: "content_block_stop", Index: 1},
				{Type: "message_delta", Delta: json.RawMessage(`{"stop_reason":"tool_use"}`)},
				{Type: "message_stop"},
			}}
			adapter, err := NewMock(backend)
			if err != nil {
				t.Fatal(err)
			}
			var script string
			for event := range adapter.Send(context.Background(), testRequest()) {
				if event.ToolCall == nil {
					continue
				}
				var input struct{ Script string `json:"script"` }
				if err := json.Unmarshal(event.ToolCall.Input, &input); err != nil {
					t.Fatal(err)
				}
				script = input.Script
			}
			if script != string(reply) {
				t.Fatalf("tool JSON script differs from model bytes: got %q want %q", script, reply)
			}
			out := filepath.Join(t.TempDir(), "result")
			cmd := exec.Command(shell, "-c", script)
			cmd.Env = append(os.Environ(), "BYTE_PACK_OUT="+out)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("execute ycode tool script: %v: %s", err, output)
			}
			got, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			want, err := hex.DecodeString(wantHex)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("output bytes = %x, want %x", got, want)
			}
		})
	}
}
