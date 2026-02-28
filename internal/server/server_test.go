package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type fakeExecutor struct {
	output string
	err    error

	gotDeviceID string
	gotMessages []string
}

func (f *fakeExecutor) Run(_ context.Context, deviceID, message string) (string, error) {
	f.gotDeviceID = deviceID
	f.gotMessages = append(f.gotMessages, message)
	return f.output, f.err
}

func TestHandleWS_MissingDeviceID(t *testing.T) {
	exec := &fakeExecutor{}
	srv := NewWithExecutor(":0", exec)
	r := srv.Engine()

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "deviceId is required") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestHandleWS_InvalidJSONPayload(t *testing.T) {
	exec := &fakeExecutor{output: `prefix {"result":"ok"} suffix`}
	srv := NewWithExecutor(":0", exec)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?deviceId=device-1"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("not-json")); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}

	if string(message) != "invalid json payload" {
		t.Fatalf("expected websocket message %q, got %q", "invalid json payload", string(message))
	}
}

func TestHandleWS_ExecutorErrorOnlyLogs(t *testing.T) {
	exec := &fakeExecutor{output: `prefix {"partial":true} suffix`, err: errors.New("boom")}
	srv := NewWithExecutor(":0", exec)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?deviceId=device-1"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"message":"hello"}`)); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatalf("expected no websocket message on executor error")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestHandleWS_SuccessAndKeepConnection(t *testing.T) {
	exec := &fakeExecutor{output: `before {"result":"ok"} after`}
	srv := NewWithExecutor(":0", exec)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?deviceId=device-1"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"message":"hello"}`)); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}
	if string(message) != `{"result":"ok"}` {
		t.Fatalf("expected websocket message %q, got %q", `{"result":"ok"}`, string(message))
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"message":"world"}`)); err != nil {
		t.Fatalf("write second websocket message: %v", err)
	}

	_, secondMessage, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read second websocket message: %v", err)
	}
	if string(secondMessage) != `{"result":"ok"}` {
		t.Fatalf("expected second websocket message %q, got %q", `{"result":"ok"}`, string(secondMessage))
	}

	if exec.gotDeviceID != "device-1" {
		t.Fatalf("executor called with unexpected deviceID=%q", exec.gotDeviceID)
	}
	if len(exec.gotMessages) != 2 || exec.gotMessages[0] != "hello" || exec.gotMessages[1] != "world" {
		t.Fatalf("executor called with unexpected messages: %#v", exec.gotMessages)
	}
}
