package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
)

// Transport is the abstraction for MCP communication channels.
type Transport interface {
	Send(ctx context.Context, data []byte) error
	Receive() <-chan []byte
	Close() error
}

// StdioTransport communicates via JSON-RPC over stdin/stdout of a subprocess.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	recv   chan []byte
	done   chan struct{}
	once   sync.Once
}

// NewStdioTransport creates a transport that launches and communicates with a subprocess.
func NewStdioTransport(cmd *exec.Cmd) (*StdioTransport, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp: start process: %w", err)
	}

	t := &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		recv:   make(chan []byte, 64),
		done:   make(chan struct{}),
	}
	go t.readLoop()
	return t, nil
}

func (t *StdioTransport) readLoop() {
	defer close(t.recv)
	scanner := bufio.NewScanner(t.stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		cp := make([]byte, len(line))
		copy(cp, line)
		select {
		case t.recv <- cp:
		case <-t.done:
			return
		}
	}
}

func (t *StdioTransport) Send(_ context.Context, data []byte) error {
	data = append(bytes.TrimRight(data, "\n"), '\n')
	_, err := t.stdin.Write(data)
	return err
}

func (t *StdioTransport) Receive() <-chan []byte { return t.recv }

func (t *StdioTransport) Close() error {
	t.once.Do(func() { close(t.done) })
	t.stdin.Close()
	return t.cmd.Wait()
}

// HTTPTransport communicates via HTTP POST (Streamable HTTP transport).
type HTTPTransport struct {
	url    string
	client *http.Client
	recv   chan []byte
	done   chan struct{}
	once   sync.Once
}

// NewHTTPTransport creates a transport that communicates with an MCP server over HTTP.
func NewHTTPTransport(url string) *HTTPTransport {
	return &HTTPTransport{
		url:    url,
		client: &http.Client{},
		recv:   make(chan []byte, 64),
		done:   make(chan struct{}),
	}
}

func (t *HTTPTransport) Send(ctx context.Context, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mcp: HTTP %d: %s", resp.StatusCode, body)
	}

	select {
	case t.recv <- body:
	case <-t.done:
	}
	return nil
}

func (t *HTTPTransport) Receive() <-chan []byte { return t.recv }

func (t *HTTPTransport) Close() error {
	t.once.Do(func() { close(t.done) })
	return nil
}

// InMemoryTransport is a pair of connected transports for testing.
type InMemoryTransport struct {
	sendCh chan []byte
	recvCh chan []byte
	done   chan struct{}
	once   sync.Once
}

// NewInMemoryTransport creates a connected pair of transports for testing.
func NewInMemoryTransport() (*InMemoryTransport, *InMemoryTransport) {
	ch1 := make(chan []byte, 64)
	ch2 := make(chan []byte, 64)
	done := make(chan struct{})
	a := &InMemoryTransport{sendCh: ch1, recvCh: ch2, done: done}
	b := &InMemoryTransport{sendCh: ch2, recvCh: ch1, done: done}
	return a, b
}

func (t *InMemoryTransport) Send(_ context.Context, data []byte) error {
	cp := make([]byte, len(data))
	copy(cp, data)
	select {
	case t.sendCh <- cp:
		return nil
	case <-t.done:
		return fmt.Errorf("mcp: transport closed")
	}
}

func (t *InMemoryTransport) Receive() <-chan []byte { return t.recvCh }

func (t *InMemoryTransport) Close() error {
	t.once.Do(func() { close(t.done) })
	return nil
}

// sendJSON marshals v and sends it over the transport.
func sendJSON(ctx context.Context, t Transport, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return t.Send(ctx, data)
}
