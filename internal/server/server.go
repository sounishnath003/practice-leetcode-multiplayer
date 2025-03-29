package server

import (
	"fmt"
	"html/template"
	"log"
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
	srv.HandleFunc("GET /", IndexHandler)
	srv.HandleFunc("GET /api/healthz", MiddlewareChain(HealthHandler, LoggerMiddleware()))
	srv.HandleFunc("POST /api/search", MiddlewareChain(SearchQuestionHandler, LoggerMiddleware()))

	// Serve the static assets
	staticFileServer := http.FileServer(http.Dir("./templates"))
	srv.Handle("GET /static/", http.StripPrefix("/static/", staticFileServer))

	s.Co.Lo.Printf("trying to start the server on http://0.0.0.0:%d\n", s.Co.Port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Co.Port), DefaultMiddlwareTracker(srv))
}

// ParseTemplates parses all template files in the specified directory and returns a compiled template.
func ParseTemplates(templateDirPattern string) (*template.Template, error) {
	log.Printf("parsing the templates from template directory %s", templateDirPattern)
	tmpl, err := template.ParseGlob(templateDirPattern)
	return tmpl, err
}
