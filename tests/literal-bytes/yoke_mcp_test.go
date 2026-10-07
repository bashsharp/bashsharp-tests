package mcp

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
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	_ "github.com/qiangli/coreutils/cmds/all"
)

type literalCase struct{ name, wantHex string }

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

// Overlaid into yoke/mcp to drive the actual MCP server and run_tool handler.
func TestSprint381LiteralBytesRunTool(t *testing.T) {
	root := os.Getenv("LITERAL_BYTES_ROOT")
	shell := os.Getenv("BASH_SHELL_BIN")
	if root == "" || shell == "" {
		t.Fatal("LITERAL_BYTES_ROOT and BASH_SHELL_BIN are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	if _, err := NewServer("literal-bytes", "test").Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "literal-bytes-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
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
			dir := t.TempDir()
			scriptPath := filepath.Join(dir, "reply.bsh")
			resultPath := filepath.Join(dir, "result")
			response, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
				Name:      "run_tool",
				Arguments: RunToolInput{Name: "tee", Args: []string{scriptPath}, Stdin: string(reply), Dir: dir},
			})
			if err != nil {
				t.Fatal(err)
			}
			var run RunToolOutput
			encoded, err := json.Marshal(response.StructuredContent)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &run); err != nil {
				t.Fatal(err)
			}
			if run.ExitCode != 0 {
				t.Fatalf("run_tool tee exit=%d stderr=%q", run.ExitCode, run.Stderr)
			}
			stored, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(stored) != string(reply) {
				t.Fatalf("run_tool reply bytes = %x, want %x", stored, reply)
			}
			cmd := exec.Command(shell, "-c", string(stored))
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "BYTE_PACK_OUT="+resultPath)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("execute MCP-replayed script: %v: %s", err, output)
			}
			got, err := os.ReadFile(resultPath)
			if err != nil {
				t.Fatal(err)
			}
			want, err := hex.DecodeString(wantHex)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("result bytes = %x, want %x", got, want)
			}
		})
	}
}
