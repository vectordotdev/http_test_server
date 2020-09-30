package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type key int

const (
	requestIDKey key = 0
)

var (
	healthy int32
)

type ServerOptions struct {
	RateLimiter Middleware
	Latency     Middleware
	Error       Middleware
}

type Server struct {
	server *http.Server
	logger *log.Logger

	quit chan (struct{})

	statisticsMiddleware *statisticsMiddleware
}

type Middleware interface {
	WrapHTTP(http.Handler) http.Handler
}

func (s *Server) Listen(listener net.Listener) {
	// Print debug output on an interval. This helps with providing insight
	// into activity without saturating IO.
	ticker := time.NewTicker(5 * time.Second)
	s.quit = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Printf("Received %v messages across %v requests", s.statisticsMiddleware.MessageCount(), s.statisticsMiddleware.RequestCount())
			case <-s.quit:
				ticker.Stop()
				return
			}
		}
	}()

	s.logger.Println("Server is ready to handle requests at", listener.Addr().String())
	atomic.StoreInt32(&healthy, 1)
	if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
		s.logger.Fatalf("Could not listen on %s: %v\n", s.server.Addr, err)
	}
}

func (s *Server) Statistics() Statistics {
	return s.statisticsMiddleware.Statistics()
}

func (s *Server) Shutdown(ctx context.Context) error {
	close(s.quit)
	s.server.SetKeepAlivesEnabled(false)
	return s.server.Shutdown(ctx)
}

func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintln(w, "")
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&healthy) == 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

func WithLatency(latency Middleware) func(*ServerOptions) {
	return func(s *ServerOptions) {
		s.Latency = latency
	}
}

func WithRateLimiter(limiter Middleware) func(*ServerOptions) {
	return func(s *ServerOptions) {
		s.RateLimiter = limiter
	}
}

func WithError(errorer Middleware) func(*ServerOptions) {
	return func(s *ServerOptions) {
		s.Error = errorer
	}
}

func NewServer(opts ...func(*ServerOptions)) *Server {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")

	router := http.NewServeMux()

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	httpServer := &http.Server{
		Handler:      tracing(nextRequestID)(logging(logger)(router)),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	server := Server{
		server: httpServer,
		logger: logger,

		statisticsMiddleware: newStatisticsMiddleware(),
	}

	errorExpressionMiddleware, err := NewErrorExpressionMiddleware("false")
	if err != nil {
		panic(err) // should never happen
	}

	serverOptions := ServerOptions{
		RateLimiter: &RateLimiterNone{},
		Latency:     NewLatencyMiddlewareNormal(time.Duration(0), time.Duration(0)),
		Error:       errorExpressionMiddleware,
	}

	for _, opt := range opts {
		opt(&serverOptions)
	}

	var indexHandler http.Handler = http.HandlerFunc(server.Index)
	indexHandler = serverOptions.Latency.WrapHTTP(indexHandler)
	indexHandler = serverOptions.Error.WrapHTTP(indexHandler)
	indexHandler = serverOptions.RateLimiter.WrapHTTP(indexHandler)
	indexHandler = server.statisticsMiddleware.WrapHTTP(indexHandler)
	router.Handle("/", indexHandler)

	router.HandleFunc("/_health", server.Health)

	return &server
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
