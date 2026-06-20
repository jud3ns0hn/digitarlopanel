package api

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// terminalMessage is the client->server control envelope. Output frames sent
// back to the client are raw text.
type terminalMessage struct {
	Type string `json:"t"`            // "i" = input, "r" = resize
	Data string `json:"d,omitempty"`  // input payload
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// handleTerminal bridges a websocket to a PTY-backed login shell. It is mounted
// admin-only and every session open is audited because it grants full shell
// access with the panel's privileges.
func (s *Server) handleTerminal(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	s.audit(c, "terminal_open", c.ClientIP())

	shell := "/bin/bash"
	if _, err := os.Stat(shell); err != nil {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("failed to start shell: "+err.Error()))
		return
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	// PTY output -> websocket.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if werr := conn.WriteMessage(websocket.TextMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				conn.Close()
				return
			}
		}
	}()

	// websocket -> PTY (input and resize control messages).
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg terminalMessage
		if json.Unmarshal(data, &msg) == nil && msg.Type != "" {
			switch msg.Type {
			case "i":
				if _, err := ptmx.Write([]byte(msg.Data)); err != nil {
					return
				}
			case "r":
				_ = pty.Setsize(ptmx, &pty.Winsize{Cols: msg.Cols, Rows: msg.Rows})
			}
			continue
		}
		// Fallback: treat raw frame as input.
		if _, err := ptmx.Write(data); err != nil {
			return
		}
	}
}
