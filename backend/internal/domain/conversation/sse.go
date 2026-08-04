package conversation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SSEWriter struct {
	w           http.ResponseWriter
	flusher     http.Flusher
	heartbeatAt time.Time
}

func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	flusher, _ := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	return &SSEWriter{w: w, flusher: flusher, heartbeatAt: time.Now().Add(15 * time.Second)}
}

func (s *SSEWriter) Send(event string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	s.heartbeatAt = time.Now().Add(15 * time.Second)
	return nil
}

// MaybeHeartbeat 15s 静默则发心跳
func (s *SSEWriter) MaybeHeartbeat() {
	if time.Now().Before(s.heartbeatAt) {
		return
	}
	_ = s.Send("heartbeat", struct{}{})
}
