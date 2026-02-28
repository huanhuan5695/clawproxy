package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
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
	log.Printf("[executor] start openclaw command session_id=%s", deviceID)
	cmd := buildOpenClawCommand(ctx, deviceID, message)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err := cmd.Run()
	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()

	if stderr != "" {
		log.Printf("[executor] openclaw command warning/error output session_id=%s stderr=%q", deviceID, stderr)
	}

	if err != nil {
		log.Printf("[executor] openclaw command failed session_id=%s err=%v", deviceID, err)
		return stdout, fmt.Errorf("run openclaw agent command: %w", err)
	}

	log.Printf("[executor] openclaw command finished session_id=%s output_bytes=%d", deviceID, len(stdout))
	return stdout, nil
}

func buildOpenClawCommand(ctx context.Context, deviceID, message string) *exec.Cmd {
	return exec.CommandContext(ctx, "openclaw", "agent", "--session-id", deviceID, "--message", message, "--json")
}

func extractJSONObject(raw string) (string, error) {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '{' {
			continue
		}

		decoder := json.NewDecoder(strings.NewReader(raw[i:]))
		decoder.UseNumber()
		var obj map[string]any
		if err := decoder.Decode(&obj); err != nil {
			continue
		}

		offset := decoder.InputOffset()
		if offset <= 0 || i+int(offset) > len(raw) {
			continue
		}

		candidate := strings.TrimSpace(raw[i : i+int(offset)])
		if !json.Valid([]byte(candidate)) {
			continue
		}

		return candidate, nil
	}

	return "", fmt.Errorf("json object not found in output")
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
	log.Printf("[server] starting websocket server addr=%s", s.addr)
	return s.Engine().Run(s.addr)
}

func (s *Server) handleWS(c *gin.Context) {
	deviceID := c.Query("deviceId")
	if deviceID == "" {
		log.Printf("[server] reject websocket request: missing deviceId")
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId is required"})
		return
	}

	log.Printf("[server] websocket upgrade requested session_id=%s", deviceID)
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[server] websocket upgrade failed session_id=%s err=%v", deviceID, err)
		return
	}
	defer conn.Close()

	log.Printf("[server] websocket connected session_id=%s", deviceID)
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[server] websocket closed/read failed session_id=%s err=%v", deviceID, err)
			return
		}

		var req wsRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			log.Printf("[server] invalid websocket json session_id=%s err=%v", deviceID, err)
			if writeErr := conn.WriteMessage(websocket.TextMessage, []byte("invalid json payload")); writeErr != nil {
				log.Printf("[server] write websocket error failed session_id=%s err=%v", deviceID, writeErr)
				return
			}
			continue
		}
		if req.Message == "" {
			log.Printf("[server] empty message in websocket payload session_id=%s", deviceID)
			if writeErr := conn.WriteMessage(websocket.TextMessage, []byte("message is required")); writeErr != nil {
				log.Printf("[server] write websocket error failed session_id=%s err=%v", deviceID, writeErr)
				return
			}
			continue
		}

		log.Printf("[server] received websocket payload session_id=%s message_len=%d", deviceID, len(req.Message))
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
		output, runErr := s.executor.Run(ctx, deviceID, req.Message)
		cancel()

		if output != "" {
			log.Printf("[server] full command output session_id=%s: %s", deviceID, output)
		}

		if runErr != nil {
			log.Printf("[server] executor failed session_id=%s err=%v", deviceID, runErr)
			continue
		}

		jsonPayload, extractErr := extractJSONObject(output)
		if extractErr != nil {
			log.Printf("[server] failed to extract json from output session_id=%s err=%v", deviceID, extractErr)
			continue
		}

		log.Printf("[server] sending extracted json over websocket session_id=%s bytes=%d", deviceID, len(jsonPayload))
		if writeErr := conn.WriteMessage(websocket.TextMessage, []byte(jsonPayload)); writeErr != nil {
			log.Printf("[server] write websocket response failed session_id=%s err=%v", deviceID, writeErr)
			return
		}
	}
}
