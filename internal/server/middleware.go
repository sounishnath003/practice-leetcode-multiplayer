package server

import (
	"log"
	"net/http"
	"time"
)

// DefaultMiddlwareTracker is a global middleware that logs every request.
func DefaultMiddlwareTracker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[DefaultMiddlwareTracker]: Started method=%s, path=%s, remoteAddr=%s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
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
