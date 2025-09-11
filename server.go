package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 16 << 20 // 16 MiB
)

type LatencyConfig struct {
	Wait time.Duration
}

type Server struct {
	router *mux.Router

	LatencyConfig LatencyConfig
}

func (s *Server) HttpPing(w http.ResponseWriter, r *http.Request) {
	time.Sleep(s.LatencyConfig.Wait)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) Echo(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade: %v", err)
		return
	}
	defer conn.Close()

	// Reader goroutine: process incoming messages
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn.SetReadLimit(maxMessageSize)
		conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived) {
					return
				}
				log.Printf("read: %v", err)
				return
			}
			// Echo
			if err := s.writeWs(conn, mt, msg); err != nil {
				return
			}
		}
	}()

	// Writer loop: periodic pings to keep the connection alive
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			// try to close gracefully
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(writeWait))
			return
		case <-ticker.C:
			_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait))
		}
	}
}

func (s *Server) writeWs(c *websocket.Conn, mt int, payload []byte) error {
	c.SetWriteDeadline(time.Now().Add(writeWait))
	return c.WriteMessage(mt, payload)
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	addr := ":8080"
	slog.Default().With("addr", addr).Info("serving http server")
	return http.ListenAndServe(addr, s.router)
}

func NewServer(router *mux.Router, latCfg LatencyConfig) *Server {
	s := &Server{
		router:        router,
		LatencyConfig: latCfg,
	}
	router.HandleFunc("/http/ping", s.HttpPing)
	router.HandleFunc("/ws", s.Echo).Methods(http.MethodGet)
	return s
}
