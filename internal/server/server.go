package server

import (
	"fmt"
	"net/http"

	"github.com/sounishnath003/practice-leetcode-multiplayer/internal/core"
)

type Server struct {
	Co *core.Core
}

// StartServer helps to start the server based on the provided configuration.
func (s *Server) StartServer() error {
	srv := http.NewServeMux()

	// Add routes
	srv.HandleFunc("GET /api/healthz", MiddlewareChain(HealthHandler, LoggerMiddleware()))
	srv.HandleFunc("GET /api/search", MiddlewareChain(SearchQuestionHandler, LoggerMiddleware()))

	s.Co.Lo.Printf("trying to start the server on http://0.0.0.0:%d\n", s.Co.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Co.Port), DefaultMiddlwareTracker(srv))
}
