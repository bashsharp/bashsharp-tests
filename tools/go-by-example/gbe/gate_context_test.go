package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestContextAdapterWaitsForCanceledHandlerReport(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	stdout := filepath.Join(t.TempDir(), "stdout")
	if err := os.WriteFile(stdout, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	closed := make(chan struct{})
	finish := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(server)
		line, err := reader.ReadString('\n')
		if err != nil || line != "GET /hello HTTP/1.1\r\n" {
			serverDone <- fmt.Errorf("request line %q: %v", line, err)
			return
		}
		for {
			line, err = reader.ReadString('\n')
			if err != nil {
				serverDone <- err
				return
			}
			if line == "\r\n" {
				break
			}
		}
		if err := os.WriteFile(stdout, []byte("server: hello handler started\n"), 0o600); err != nil {
			serverDone <- err
			return
		}
		if _, err := io.ReadAll(reader); err != nil {
			serverDone <- err
			return
		}
		close(closed)
		<-finish
		if err := os.WriteFile(stdout, []byte("server: hello handler started\nserver: context canceled\nserver: hello handler ended\n"), 0o600); err != nil {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	adapterDone := make(chan error, 1)
	go func() { adapterDone <- driveContext(client, stdout, monotonicSeconds()+2, newStopFlag()) }()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("adapter did not close the client after handler startup")
	}
	select {
	case err := <-adapterDone:
		t.Fatalf("adapter returned before the canceled handler finished: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(finish)
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-adapterDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("adapter did not return after the canceled handler report")
	}
	data, err := os.ReadFile(stdout)
	if err != nil || !strings.Contains(string(data), "server: context canceled\n") {
		t.Fatalf("handler report %q: %v", data, err)
	}
}
