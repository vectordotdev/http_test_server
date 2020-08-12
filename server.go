package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juju/ratelimit"
)

type key int

const (
	requestIDKey key = 0
)

var (
	healthy int32
)

type Server struct {
	server *http.Server
	logger *log.Logger

	quit chan (struct{})

	// artificial parameters
	latency           time.Duration
	rateLimitBucket   *ratelimit.Bucket
	rateLimitBehavior RateLimitBehavior

	// TDOO(jesse) consider moving Statistics to handler with channel to avoid
	// requests blocking each other and to drain on shutdown
	Statistics *Statistics
}

type Statistics struct {
	sync.Mutex

	ByteTotal    int64  `json:"byte_total"`
	FirstMessage string `json:"first_message"`
	LastMessage  string `json:"last_message"`
	MessageCount int64  `json:"message_count"`
	RequestCount int64  `json:"request_count"`

	Requests []*RequestStatistics `json:"requests"`
}
type RequestStatistics struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Status int       `json:"status"`
}

func (s *Server) Listen() {
	// Print debug output on an interval. This helps with providing insight
	// into activity without saturating IO.
	ticker := time.NewTicker(5 * time.Second)
	s.quit = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				availableTokens := int64(-1)
				if s.rateLimitBucket != nil {
					availableTokens = s.rateLimitBucket.Available()
				}
				log.Printf("Received %v messages across %v requests. Tokens available: %d (-1 indicates no rate limit)", s.Statistics.MessageCount, s.Statistics.RequestCount, availableTokens)
			case <-s.quit:
				ticker.Stop()
				return
			}
		}
	}()

	s.logger.Println("Server is ready to handle requests at", s.server.Addr)
	atomic.StoreInt32(&healthy, 1)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Fatalf("Could not listen on %s: %v\n", s.server.Addr, err)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	close(s.quit)
	s.server.SetKeepAlivesEnabled(false)
	return s.server.Shutdown(ctx)
}

func (s *Server) Index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// built up during request handling and statistics recorded at the end via defer
		handledRequest := &handledRequest{
			startTime: time.Now(),
		}

		defer func() {
			handledRequest.endTime = time.Now()

			go func() {
				s.recordRequest(handledRequest)
			}()
		}()

		handledRequest.contentType = r.Header.Get("Content-Type")
		handledRequest.contentLength = r.Header.Get("Content-Length")

		//
		// Handle request using test parameters
		//
		if bucket := s.rateLimitBucket; bucket != nil {
			switch s.rateLimitBehavior {
			case RateLimitBehaviorHard:
				if bucket.TakeAvailable(1) == 0 {
					handledRequest.statusCode = http.StatusTooManyRequests
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}

			case RateLimitBehaviorQueue:
				bucket.Wait(1)

			case RateLimitBehaviorNone:

			default:
				panic(fmt.Sprintf("unknown rate limit behavior: %s", s.rateLimitBehavior))
			}
		}
		time.Sleep(s.latency)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			handledRequest.statusCode = http.StatusBadRequest

			s.logger.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		handledRequest.body = body
		handledRequest.statusCode = http.StatusNoContent

		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintln(w, "")
	})
}

type handledRequest struct {
	startTime     time.Time
	endTime       time.Time
	body          []byte
	contentType   string
	contentLength string
	statusCode    int
}

func (s *Server) recordRequest(r *handledRequest) {
	s.logger.Printf("Received request: content-type: %v, content-length: %v", r.contentType, r.contentLength)

	byteLen := len(r.body)
	body := string(r.body)
	messages := []string{}

	switch r.contentType {
	// Unfortunately fluentbit does not use the proper content type when sending
	// new line delimited JSON :(
	case "application/json":
		messages = strings.Split(body, "\n")
	case "application/ndjson":
		messages = strings.Split(body, "\n")
	case "application/x-ndjson":
		messages = strings.Split(body, "\n")
	case "text/plain":
		messages = strings.Split(body, "\n")
	}

	messageCount := len(messages)
	firstMessage := ""
	lastMessage := ""
	if messageCount > 0 {
		firstMessage = messages[0]
		lastMessage = messages[messageCount-1]
	}

	s.Statistics.Lock()
	defer s.Statistics.Unlock()

	s.Statistics.RequestCount++

	s.Statistics.ByteTotal += int64(byteLen)
	s.Statistics.MessageCount += int64(messageCount)

	if s.Statistics.FirstMessage == "" {
		s.Statistics.FirstMessage = firstMessage
	}
	if lastMessage != "" {
		s.Statistics.LastMessage = lastMessage
	}

	s.Statistics.Requests = append(s.Statistics.Requests, &RequestStatistics{
		Start:  r.startTime.UTC(),
		End:    r.endTime.UTC(),
		Status: r.statusCode,
	})
}

func (s *Server) Health() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}

func WithLatency(d time.Duration) func(*Server) {
	return func(s *Server) {
		s.latency = d
	}
}

func WithRateLimit(behavior RateLimitBehavior, fillInterval time.Duration, capacity, quantum int64) func(*Server) {
	return func(s *Server) {
		s.rateLimitBehavior = behavior
		s.rateLimitBucket = ratelimit.NewBucketWithQuantum(fillInterval, capacity, quantum)
	}
}

func NewServer(address string, opts ...func(*Server)) *Server {
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")

	router := http.NewServeMux()

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	httpServer := &http.Server{
		Addr:         address,
		Handler:      tracing(nextRequestID)(logging(logger)(router)),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	server := Server{
		server: httpServer,
		logger: logger,

		rateLimitBehavior: RateLimitBehaviorNone,

		Statistics: &Statistics{},
	}

	for _, opt := range opts {
		opt(&server)
	}

	router.Handle("/", server.Index())
	router.Handle("/_health", server.Health())

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

type RateLimitBehavior string

const (
	// no rate limit
	RateLimitBehaviorNone RateLimitBehavior = "NONE"

	// returns a 429 when rate is exceeded
	RateLimitBehaviorHard RateLimitBehavior = "HARD"

	// queues request until there is available capacity
	RateLimitBehaviorQueue RateLimitBehavior = "QUEUE"
)
