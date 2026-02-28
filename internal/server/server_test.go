package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

type fakeExecutor struct {
	output string
	err    error

	gotDeviceID string
	gotMessage  string
}

func (f *fakeExecutor) Run(_ context.Context, deviceID, message string) (string, error) {
	f.gotDeviceID = deviceID
	f.gotMessage = message
	return f.output, f.err
}

func TestHandleWS_MissingQueryParams(t *testing.T) {
	exec := &fakeExecutor{}
	srv := NewWithExecutor(":0", exec)
	r := srv.Engine()

	req := httptest.NewRequest(http.MethodGet, "/ws?deviceId=device-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "deviceId and message are required") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestHandleWS_Success(t *testing.T) {
	exec := &fakeExecutor{output: "command output"}
	srv := NewWithExecutor(":0", exec)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?deviceId=device-1&message=hello"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}

	if string(message) != "command output" {
		t.Fatalf("expected websocket message %q, got %q", "command output", string(message))
	}

	if exec.gotDeviceID != "device-1" || exec.gotMessage != "hello" {
		t.Fatalf("executor called with unexpected params: deviceID=%q, message=%q", exec.gotDeviceID, exec.gotMessage)
	}
}
