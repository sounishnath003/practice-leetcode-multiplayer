package server

import (
	"context"
	"log"
	"net/http"
	"time"
)

// DefaultMiddlwareTracker is a global middleware that logs every request.
func DefaultMiddlwareTracker(next http.Handler) http.Handler {

	// Parse the templates and store
	// Add template into the Request Context to be used by All routes
	tmpl, err := ParseTemplates("templates/*.html")
	if err != nil {
		log.Printf("[ERROR]: error occurred while parsing the templates: %v", err)
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[DefaultMiddlwareTracker]: Started method=%s, path=%s, remoteAddr=%s", r.Method, r.URL.Path, r.RemoteAddr)
		ctx := context.WithValue(r.Context(), "template", tmpl)
		newCtx := r.WithContext(ctx)

		next.ServeHTTP(w, newCtx)
		log.Printf("[DefaultMiddlwareTracker]: Completed remoteAddr=%s, method=%s, path=%s in time=%v", r.RemoteAddr, r.Method, r.URL.Path, time.Since(start))
	})
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

func LoggerMiddleware() Middleware {
	return func(hf http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log.Printf("[LoggerMiddlware]: path=%s, remoteAddr=%s", r.URL.Path, r.RemoteAddr)
			hf(w, r)
		}
	}
}

func MiddlewareChain(hf http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	for _, middleware := range middlewares {
		hf = middleware(hf)
	}
	return hf
}
