package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Statistics struct {
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

// TODO(jesse) consider moving Statistics to handler with channel to avoid
// requests blocking each other and to drain on shutdown
type statisticsMiddleware struct {
	mu         sync.Mutex
	statistics *Statistics
}

func newStatisticsMiddleware() *statisticsMiddleware {
	return &statisticsMiddleware{
		statistics: &Statistics{},
	}
}

func (sm *statisticsMiddleware) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		handledRequest := &handledRequest{
			startTime:     time.Now(),
			contentType:   r.Header.Get("Content-Type"),
			contentLength: r.Header.Get("Content-Length"),
		}

		var b bytes.Buffer
		_, err := b.ReadFrom(r.Body)
		if err != nil {
			handledRequest.statusCode = http.StatusBadRequest
			http.Error(rw, "can't read body", http.StatusBadRequest)
			return
		}
		r.Body = ioutil.NopCloser(&b)
		handledRequest.body = b.Bytes()

		wrapper := &responseWriterWrapper{ResponseWriter: rw}

		next.ServeHTTP(wrapper, r)

		handledRequest.statusCode = wrapper.status
		handledRequest.endTime = time.Now()
		go func() {
			sm.recordRequest(handledRequest)
		}()
	})
}

func (sm *statisticsMiddleware) recordRequest(r *handledRequest) {
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

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.statistics.RequestCount++

	sm.statistics.ByteTotal += int64(byteLen)
	sm.statistics.MessageCount += int64(messageCount)

	if sm.statistics.FirstMessage == "" {
		sm.statistics.FirstMessage = firstMessage
	}
	if lastMessage != "" {
		sm.statistics.LastMessage = lastMessage
	}

	sm.statistics.Requests = append(sm.statistics.Requests, &RequestStatistics{
		Start:  r.startTime.UTC(),
		End:    r.endTime.UTC(),
		Status: r.statusCode,
	})
}

func (sm *statisticsMiddleware) MessageCount() int64 {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.statistics.MessageCount
}

func (sm *statisticsMiddleware) RequestCount() int64 {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.statistics.RequestCount
}

func (sm *statisticsMiddleware) Statistics() Statistics {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return *sm.statistics
}

type handledRequest struct {
	startTime     time.Time
	endTime       time.Time
	body          []byte
	contentType   string
	contentLength string
	statusCode    int
}

type responseWriterWrapper struct {
	http.ResponseWriter
	written int64
	status  int
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}
