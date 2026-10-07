package mcp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	_ "github.com/qiangli/coreutils/cmds/all"
)

// Overlaid into yoke/mcp to drive the actual MCP server and run_tool handler.
func TestSprint381LiteralBytesRunTool(t *testing.T) {
	root := os.Getenv("LITERAL_BYTES_ROOT")
	bashy := os.Getenv("BASHY_BIN")
	if root == "" || bashy == "" {
		t.Fatal("LITERAL_BYTES_ROOT and BASHY_BIN are required")
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
			dir := t.TempDir()
			scriptPath := filepath.Join(dir, "reply.bsh")
			resultPath := filepath.Join(dir, "result")
			response, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
				Name: "run_tool",
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
			cmd := exec.Command(bashy, "-c", string(stored))
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
