package server

// managed_stdio.go adds the small host-lifecycle layer used when METIS owns
// this helper. The standard MCP methods still go through mcp-go; the two
// reserved requests below are deliberately not exposed as model-callable
// tools. Keeping them in the transport layer lets the host cancel in-flight
// input before it closes the process.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	computerUseStop    = "metis-cu/stop"
	computerUseEndTurn = "metis-cu/end-turn"
)

type managedStdioSession struct {
	notifications chan mcp.JSONRPCNotification
	initialized   atomic.Bool
}

func newManagedStdioSession() *managedStdioSession {
	return &managedStdioSession{notifications: make(chan mcp.JSONRPCNotification, 16)}
}

func (s *managedStdioSession) Initialize() { s.initialized.Store(true) }

func (s *managedStdioSession) Initialized() bool { return s.initialized.Load() }

func (s *managedStdioSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return s.notifications
}

func (s *managedStdioSession) SessionID() string { return "stdio" }

type managedControlResult struct {
	Stopped bool `json:"stopped"`
	Cleaned bool `json:"cleaned"`
}

type managedStdio struct {
	ctx      context.Context
	server   *mcpserver.MCPServer
	platform platform.Platform
	session  *managedStdioSession

	writeMu sync.Mutex

	callMu   sync.Mutex
	calls    map[string]context.CancelFunc
	callWg   sync.WaitGroup
	toolSlot chan struct{}
	paused   bool
	stopped  bool

	controlMu sync.Mutex
}

func newManagedStdio(ctx context.Context, srv *mcpserver.MCPServer, plat platform.Platform) *managedStdio {
	return &managedStdio{
		ctx:      ctx,
		server:   srv,
		platform: plat,
		session:  newManagedStdioSession(),
		calls:    make(map[string]context.CancelFunc),
		toolSlot: make(chan struct{}, 1),
	}
}

func serveManagedStdio(srv *mcpserver.MCPServer, plat platform.Platform) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	managed := newManagedStdio(ctx, srv, plat)
	err := managed.listen(os.Stdin, os.Stdout)
	managed.cancelCalls()
	managed.callWg.Wait()
	return err
}

func (s *managedStdio) listen(stdin io.Reader, stdout io.Writer) error {
	if err := s.server.RegisterSession(s.ctx, s.session); err != nil {
		return fmt.Errorf("register session: %w", err)
	}
	defer s.server.UnregisterSession(s.ctx, s.session.SessionID())
	ctx := s.server.WithContext(s.ctx, s.session)

	go func() {
		for {
			select {
			case <-s.session.notifications:
			case <-ctx.Done():
				return
			}
		}
	}()

	reader := bufio.NewReader(stdin)
	for {
		line, err := reader.ReadString('\n')
		if text := strings.TrimSpace(line); text != "" {
			if handleErr := s.handleLine(ctx, []byte(text), stdout); handleErr != nil {
				return handleErr
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func (s *managedStdio) handleLine(ctx context.Context, raw []byte, stdout io.Writer) error {
	var envelope struct {
		Method string        `json:"method"`
		ID     mcp.RequestId `json:"id"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return s.writeResponse(s.server.HandleMessage(ctx, raw), stdout)
	}

	switch envelope.Method {
	case computerUseStop, computerUseEndTurn:
		if envelope.ID.IsNil() {
			go s.control(envelope.Method == computerUseStop)
			return nil
		}
		result := s.control(envelope.Method == computerUseStop)
		return s.writeResponse(mcp.NewJSONRPCResultResponse(envelope.ID, result), stdout)
	case "tools/call":
		if envelope.ID.IsNil() {
			return s.writeResponse(s.server.HandleMessage(ctx, raw), stdout)
		}
		s.startToolCall(ctx, raw, envelope.ID, stdout)
		return nil
	default:
		return s.writeResponse(s.server.HandleMessage(ctx, raw), stdout)
	}
}

func (s *managedStdio) startToolCall(parent context.Context, raw []byte, id mcp.RequestId, stdout io.Writer) {
	key := id.String()
	callCtx, cancel := context.WithCancel(parent)

	s.callMu.Lock()
	if s.paused || s.stopped {
		s.callMu.Unlock()
		_ = s.writeResponse(mcp.NewJSONRPCError(id, mcp.INVALID_REQUEST, "computer-use helper is stopping", nil), stdout)
		cancel()
		return
	}
	if _, exists := s.calls[key]; exists {
		s.callMu.Unlock()
		_ = s.writeResponse(mcp.NewJSONRPCError(id, mcp.INVALID_REQUEST, "duplicate request id", nil), stdout)
		cancel()
		return
	}
	s.calls[key] = cancel
	s.callWg.Add(1)
	s.callMu.Unlock()

	go func() {
		defer s.callWg.Done()
		defer func() {
			s.callMu.Lock()
			delete(s.calls, key)
			s.callMu.Unlock()
		}()

		select {
		case s.toolSlot <- struct{}{}:
			defer func() { <-s.toolSlot }()
		case <-callCtx.Done():
			_ = s.writeResponse(mcp.NewJSONRPCError(id, mcp.INVALID_REQUEST, callCtx.Err().Error(), nil), stdout)
			return
		}

		if response := s.server.HandleMessage(callCtx, raw); response != nil {
			_ = s.writeResponse(response, stdout)
		}
	}()
}

func (s *managedStdio) control(stop bool) managedControlResult {
	s.controlMu.Lock()
	defer s.controlMu.Unlock()

	s.callMu.Lock()
	s.paused = true
	if stop {
		s.stopped = true
	}
	cancels := make([]context.CancelFunc, 0, len(s.calls))
	for _, cancel := range s.calls {
		cancels = append(cancels, cancel)
	}
	s.callMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	s.callWg.Wait()

	cleaned := true
	if stop && s.platform != nil {
		cleaned = s.platform.Close() == nil
	}
	if !stop {
		s.callMu.Lock()
		s.paused = false
		s.callMu.Unlock()
	}
	return managedControlResult{Stopped: stop, Cleaned: cleaned}
}

func (s *managedStdio) cancelCalls() {
	s.callMu.Lock()
	s.paused = true
	cancels := make([]context.CancelFunc, 0, len(s.calls))
	for _, cancel := range s.calls {
		cancels = append(cancels, cancel)
	}
	s.callMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (s *managedStdio) writeResponse(response mcp.JSONRPCMessage, stdout io.Writer) error {
	if response == nil {
		return nil
	}
	payload, err := json.Marshal(response)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err = fmt.Fprintf(stdout, "%s\n", payload)
	return err
}
