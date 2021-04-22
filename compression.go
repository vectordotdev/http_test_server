package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
)

type compressionMiddleware struct{}

func NewCompressionMiddleware() *compressionMiddleware {
	return &compressionMiddleware{}
}

func (cm *compressionMiddleware) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var reader io.ReadCloser

		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			var err error
			reader, err = gzip.NewReader(r.Body)
			if err != nil {
				panic(fmt.Sprintf("could not read gzip body: %s", err))
			}
		default:
			reader = r.Body
		}

		r.Body = reader

		next.ServeHTTP(rw, r)
	})
}
