package provider

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	api "github.com/qiangli/ycode/internal/api"
)

type literalCase struct{ name, wantHex string }

func TestParseLiteralCasesRejectsMalformedInput(t *testing.T) {
	var rows []string
	for i := 0; i < 15; i++ {
		rows = append(rows, fmt.Sprintf("case%d\t00", i))
	}
	for name, mutate := range map[string]func([]string){
		"duplicate":   func(r []string) { r[14] = "case0\t00" },
		"bad hex":     func(r []string) { r[14] = "case14\tzz" },
		"extra field": func(r []string) { r[14] += "\textra" },
	} {
		t.Run(name, func(t *testing.T) {
			copyRows := append([]string(nil), rows...)
			mutate(copyRows)
			if _, err := parseLiteralCases([]byte(strings.Join(copyRows, "\n"))); err == nil {
				t.Fatal("accepted malformed cases.tsv")
			}
		})
	}
}

func parseLiteralCases(data []byte) ([]literalCase, error) {
	var cases []literalCase
	seen := map[string]bool{}
	s := bufio.NewScanner(strings.NewReader(string(data)))
	for line := 1; s.Scan(); line++ {
		row := s.Text()
		if strings.TrimSpace(row) == "" || strings.HasPrefix(row, "#") {
			continue
		}
		fields := strings.Split(row, "\t")
		if len(fields) != 2 || fields[0] == "" || fields[1] == "" {
			return nil, fmt.Errorf("line %d: expected name and hex fields", line)
		}
		if seen[fields[0]] {
			return nil, fmt.Errorf("line %d: duplicate case %q", line, fields[0])
		}
		fields[1] = strings.TrimSpace(fields[1])
		if fields[1] == "" {
			return nil, fmt.Errorf("line %d: empty hex field", line)
		}
		if _, err := hex.DecodeString(fields[1]); err != nil {
			return nil, fmt.Errorf("line %d: invalid hex: %w", line, err)
		}
		seen[fields[0]] = true
		cases = append(cases, literalCase{fields[0], fields[1]})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(cases) != 15 {
		return nil, fmt.Errorf("expected 15 cases, got %d", len(cases))
	}
	return cases, nil
}

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
	cases, err := parseLiteralCases(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		name, wantHex := tc.name, tc.wantHex
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
				var input struct {
					Script string `json:"script"`
				}
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
