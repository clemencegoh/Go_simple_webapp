package server

import (
	"Go_simple_webapp/handlers"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"time"
)

type key int

const (
	requestIDKey key = 0
)

type Server struct {
	Logger  *log.Logger
	Healthy int32
}

func (s *Server) healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if atomic.LoadInt32(&s.Healthy) == 1 {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}
func (s *Server) welcome() http.Handler{
	return http.HandlerFunc(handlers.HomeHandler)
}




func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", s.healthz())
	mux.Handle("/", s.welcome())

	return mux
}

func (s *Server) ListenAndServe(addr string) {
	s.Logger.Println("server is starting...")

	nextRequestID := func() string {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	handler := s.routes()

	srv := &http.Server{
		Addr:         addr,
		Handler:      compose(logging(s.Logger), tracing(nextRequestID))(handler),
		ErrorLog:     s.Logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		s.Logger.Println("server is shutting down...")
		atomic.StoreInt32(&s.Healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			s.Logger.Fatalf("could not gracefully shutdown the server: %v", err)
		}
		close(done)
	}()

	s.Logger.Println("server is ready to handle requests at", addr)
	atomic.StoreInt32(&s.Healthy, 1)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.Logger.Fatalf("could not listen on %s: %v", addr, err)
	}

	<-done
	s.Logger.Println("server stopped")
}

func compose(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := next
		for _, middleware := range middlewares {
			handler = middleware(handler)
		}
		return handler
	}
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/healthz" {
				defer func() {
					requestID, ok := req.Context().Value(requestIDKey).(string)
					if !ok {
						requestID = "unknown"
					}
					logger.Println(requestID, req.Method, req.URL.Path, req.RemoteAddr, req.UserAgent())
				}()
			}
			next.ServeHTTP(w, req)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			const requestIDName = "Request-Id"
			requestID := req.Header.Get(requestIDName)
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(req.Context(), requestIDKey, requestID)
			w.Header().Set(requestIDName, requestID)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}

func cors(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization")
			// for preflight
			if req.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
				w.WriteHeader(http.StatusOK)
			} else {
				next.ServeHTTP(w, req)
			}
		})
	}
}
