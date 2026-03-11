package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"
)

type commandRequest struct {
	Type            string         `json:"type"`
	Payload         map[string]any `json:"payload"`
	ClientRequestID string         `json:"clientRequestId"`
}

type commandResponse struct {
	Accepted  bool          `json:"accepted"`
	CommandID string        `json:"commandId,omitempty"`
	Error     *commandError `json:"error,omitempty"`
}

type commandError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handleCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, commandResponse{
			Accepted: false,
			Error:    &commandError{Code: "INVALID_JSON", Message: "invalid JSON body"},
		})
		return
	}

	if req.Type == "" {
		writeJSON(w, http.StatusBadRequest, commandResponse{
			Accepted: false,
			Error:    &commandError{Code: "INVALID_COMMAND", Message: "missing command type"},
		})
		return
	}

	if req.Payload == nil {
		req.Payload = map[string]any{}
	}

	resp := commandResponse{Accepted: true, CommandID: fmt.Sprintf("cmd_%d", time.Now().UnixNano())}

	err := s.executeCommand(req)
	if err != nil {
		resp.Accepted = false
		resp.Error = &commandError{Code: "COMMAND_REJECTED", Message: err.Error()}
		writeJSON(w, http.StatusConflict, resp)
		return
	}

	s.publish(SSEEvent{Type: "command_accepted", Data: map[string]any{"type": req.Type, "clientRequestId": req.ClientRequestID}})
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) executeCommand(req commandRequest) error {
	switch req.Type {
	case "enqueue":
		query, _ := req.Payload["query"].(string)
		if query == "" {
			return errors.New("enqueue command requires payload.query")
		}
		track, err := s.svc.Resolve(context.Background(), query)
		if err != nil {
			return err
		}
		err = s.svc.Enqueue(track)
		if err != nil {
			return err
		}
		return nil
	case "pause":
		s.svc.Pause()
		return nil
	case "resume":
		s.svc.Resume()
		return nil
	case "skip":
		s.svc.Skip()
		return nil
	case "seek":
		raw, ok := req.Payload["positionSec"]
		if !ok {
			return errors.New("seek command requires payload.positionSec")
		}
		positionSec, ok := parseNumber(raw)
		if !ok || math.IsNaN(positionSec) || math.IsInf(positionSec, 0) {
			return errors.New("seek payload.positionSec must be a finite number")
		}
		if positionSec < 0 {
			positionSec = 0
		}
		err := s.svc.Seek(time.Duration(positionSec * float64(time.Second)))
		if err != nil {
			return err
		}
		return nil
	case "stop":
		s.svc.Stop()
		return nil
	case "refresh":
		s.publish(SSEEvent{Type: "snapshot"})
		return nil
	default:
		return fmt.Errorf("unsupported command type: %s", req.Type)
	}
}

func parseNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

func (s *Server) publish(event SSEEvent) {
	if s.broker != nil {
		s.broker.Publish(event)
	}
}
