package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"clawproxy/internal/auth"
	"github.com/gorilla/websocket"
)

const testJWTSecret = "unit-test-secret"

func mustCreateToken(t *testing.T) string {
	t.Helper()
	tokenString, err := auth.GenerateToken([]byte(testJWTSecret), "device-1", time.Hour)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return tokenString
}

func dialWS(t *testing.T, wsURL, token string) *websocket.Conn {
	t.Helper()
	headers := http.Header{}
	if token != "" {
		headers.Set("Authorization", token)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}

	return conn
}

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
	srv := NewWithExecutor(":0", testJWTSecret, exec)
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

func TestHandleWS_MissingToken(t *testing.T) {
	exec := &fakeExecutor{}
	srv := NewWithExecutor(":0", testJWTSecret, exec)
	r := srv.Engine()

	req := httptest.NewRequest(http.MethodGet, "/ws?deviceId=device-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	if !strings.Contains(w.Body.String(), "TOKEN_REQUIRED") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestHandleWS_InvalidToken(t *testing.T) {
	exec := &fakeExecutor{}
	srv := NewWithExecutor(":0", testJWTSecret, exec)
	r := srv.Engine()

	req := httptest.NewRequest(http.MethodGet, "/ws?deviceId=device-1", nil)
	req.Header.Set("Authorization", "bad-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	if !strings.Contains(w.Body.String(), "INVALID_TOKEN") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestHandleWS_InvalidJSONPayload(t *testing.T) {
	exec := &fakeExecutor{output: `prefix {"result":"ok"} suffix`}
	srv := NewWithExecutor(":0", testJWTSecret, exec)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?deviceId=device-1"
	conn := dialWS(t, wsURL, mustCreateToken(t))
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
	srv := NewWithExecutor(":0", testJWTSecret, exec)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?deviceId=device-1"
	conn := dialWS(t, wsURL, mustCreateToken(t))
	var err error
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
	srv := NewWithExecutor(":0", testJWTSecret, exec)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?deviceId=device-1"
	conn := dialWS(t, wsURL, mustCreateToken(t))
	var err error
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

func TestHandleWS_InvalidToken_WebSocketHandshake(t *testing.T) {
	exec := &fakeExecutor{}
	srv := NewWithExecutor(":0", testJWTSecret, exec)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?deviceId=device-1"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Authorization": []string{"bad-token"}})
	if err == nil {
		t.Fatal("expected websocket handshake failure with invalid token")
	}
	if resp == nil {
		t.Fatal("expected http response on handshake failure")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestHandleWS_MissingToken_WebSocketHandshake(t *testing.T) {
	exec := &fakeExecutor{}
	srv := NewWithExecutor(":0", testJWTSecret, exec)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?deviceId=device-1"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected websocket handshake failure with missing token")
	}
	if resp == nil {
		t.Fatal("expected http response on handshake failure")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	if !strings.Contains(resp.Request.URL.String(), "/ws?deviceId=device-1") {
		t.Fatalf("unexpected request url: %s", resp.Request.URL.String())
	}

	if _, err := url.Parse(resp.Request.URL.String()); err != nil {
		t.Fatalf("parse request url: %v", err)
	}
}
