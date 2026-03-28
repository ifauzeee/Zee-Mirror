package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"zee-mirror/handlers"
	"zee-mirror/internal/downloader"
	"zee-mirror/internal/router"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	Service    *handlers.BotService
	Router     *router.Router
	httpServer *http.Server
	Port       int
}

func NewServer(service *handlers.BotService, port int) *Server {
	return &Server{
		Service: service,
		Port:    port,
	}
}

func (s *Server) SetRouter(r *router.Router) {
	s.Router = r
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		version := "unknown"
		var err error
		if engine, ok := s.Service.TaskManager.Aria2Engine.(*downloader.Aria2Engine); ok && engine.RPC != nil {
			version, err = engine.RPC.GetVersion()
		}

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":    "error",
				"component": "aria2",
				"error":     err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":        "ok",
			"aria2_version": version,
		})
	})

	mux.Handle("/metrics", promhttp.Handler())

	if s.Service.Config.UseWebhook {
		mux.HandleFunc("/api/telegram/webhook", s.handleWebhook)
		slog.Info("Webhook endpoint registered at /api/telegram/webhook")
	}

	addr := fmt.Sprintf(":%d", s.Port)
	slog.Info("API server starting", "addr", addr)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("API server failed", "error", err)
		}
	}()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		slog.Info("Shutting down API server...")
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
