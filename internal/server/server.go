package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type CommandExecutor interface {
	Run(ctx context.Context, deviceID, message string) (string, error)
}

type OpenClawExecutor struct{}

type wsRequest struct {
	Message string `json:"message"`
}

func (e OpenClawExecutor) Run(ctx context.Context, deviceID, message string) (string, error) {
	cmd := buildOpenClawCommand(ctx, deviceID, message)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run openclaw agent command: %w, output: %s", err, string(out))
	}

	return string(out), nil
}

func buildOpenClawCommand(ctx context.Context, deviceID, message string) *exec.Cmd {
	return exec.CommandContext(ctx, "openclaw", "agent", "--session-id", deviceID, "--message", message, "--thinking", "medium")
}

type Server struct {
	addr     string
	executor CommandExecutor
	upgrader websocket.Upgrader
}

func New(addr string) *Server {
	return &Server{
		addr:     addr,
		executor: OpenClawExecutor{},
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func NewWithExecutor(addr string, executor CommandExecutor) *Server {
	s := New(addr)
	s.executor = executor
	return s
}

func (s *Server) Engine() *gin.Engine {
	r := gin.Default()
	r.GET("/ws", s.handleWS)
	return r
}

func (s *Server) Run() error {
	return s.Engine().Run(s.addr)
}

func (s *Server) handleWS(c *gin.Context) {
	deviceID := c.Query("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId is required"})
		return
	}

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	_, payload, err := conn.ReadMessage()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("read websocket message failed"))
		return
	}

	var req wsRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("invalid json payload"))
		return
	}
	if req.Message == "" {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("message is required"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	output, err := s.executor.Run(ctx, deviceID, req.Message)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}

	_ = conn.WriteMessage(websocket.TextMessage, []byte(output))
}
