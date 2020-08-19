package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type parameters struct {
	LatencyMean *string `json:"latency_mean"`

	RateLimitBehavior           *string `json:"rate_limit_behavior"`
	RateLimitBucketFillInterval *string `json:"rate_limit_bucket_fill_interval"`
	RateLimitBucketCapacity     *int64  `json:"rate_limit_bucket_capaticy"`
	RateLimitBucketQuauntum     *int64  `json:"rate_limit_bucket_quantum"`
	RateLimitHardStatusCode     *int    `json:"rate_limit_hard_status_code"`
}

var rootCmd = &cobra.Command{
	Use:   "http_test_server",
	Short: "A simple HTTP server useful for testing.",
	Long:  "This is used in Vector's test harness to test and benchmark HTTP performance. https://github.com/timberio/vector-test-harness",
	RunE: func(cmd *cobra.Command, args []string) error {
		summaryPath := viper.GetString("summary-path")
		os.Remove(summaryPath)

		parametersPath := viper.GetString("parameters-path")

		opts := []func(*ServerOptions){}

		parameters := &parameters{}

		if latency := viper.GetDuration("latency-mean"); latency > 0 {
			s := latency.String()
			parameters.LatencyMean = &s
			opts = append(opts, WithLatency(NewLatencyMiddlewareStatic(latency)))
		}

		if behavior := viper.GetString("rate-limit-behavior"); behavior != "NONE" {
			var (
				fillInterval = viper.GetDuration("rate-limit-bucket-fill-interval")
				capacity     = viper.GetInt64("rate-limit-bucket-capacity")
				quantum      = viper.GetInt64("rate-limit-bucket-quantum")
			)

			if fillInterval <= 0 {
				return fmt.Errorf("--rate-limit-bucket-fill-interval must be > 0 if --rate-limit-behavior is set to not NONE")
			}
			if capacity <= 0 {
				return fmt.Errorf("--rate-limit-bucket-capacity must be > 0 if --rate-limit-behavior is set to not NONE")
			}
			if quantum <= 0 {
				return fmt.Errorf("--rate-limit-bucket-quantum must be > 0 if --rate-limit-behavior is set to not NONE")
			}

			var rateLimiter RateLimiter
			switch behavior {
			case "HARD":
				code := viper.GetInt("rate-limit-hard-status-code")
				rateLimiter = NewRateLimiterHard(fillInterval, capacity, quantum, code)
				parameters.RateLimitHardStatusCode = &code
			case "QUEUE":
				rateLimiter = NewRateLimiterQueue(fillInterval, capacity, quantum)
			case "CLOSE":
				rateLimiter = NewRateLimiterClose(fillInterval, capacity, quantum)
			default:
				return fmt.Errorf("unknown rate-limit-behavior value: %s", behavior)
			}

			parameters.RateLimitBehavior = &behavior
			parameters.RateLimitBucketFillInterval = func() *string {
				s := fillInterval.String()
				return &s
			}()
			parameters.RateLimitBucketCapacity = &capacity
			parameters.RateLimitBucketQuauntum = &quantum

			opts = append(opts, WithRateLimiter(rateLimiter))
		}

		if parametersPath != "" {
			b, err := json.Marshal(parameters)
			if err != nil {
				log.Fatal(err)
			}

			err = ioutil.WriteFile(parametersPath, b, 0644)
			if err != nil {
				log.Fatal(err)
			}
		}

		server := NewServer(viper.GetString("address"), opts...)

		done := make(chan struct{})

		go func() {
			var gracefulStop = make(chan os.Signal, 1)
			signal.Notify(gracefulStop, syscall.SIGTERM)
			signal.Notify(gracefulStop, syscall.SIGINT)

			sig := <-gracefulStop
			log.Printf("Caught sig: %+v", sig)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				// Error from closing listeners, or context timeout:
				log.Printf("could not gracefully shutdown the server: %v\n", err)
				return
			}

			sBytes, err := json.Marshal(server.Statistics())
			if err != nil {
				log.Fatal(err)
			}

			err = ioutil.WriteFile(summaryPath, sBytes, 0644)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Wrote activity summary to %s\n", summaryPath)

			close(done)
		}()

		go func() {
			server.Listen()
		}()

		<-done

		return nil
	},
}

func main() {
	rootCmd.PersistentFlags().StringP("address", "a", "0.0.0.0:8080", "the address to bind to")

	// TODO(jesse) add variance parameter(s)
	rootCmd.PersistentFlags().DurationP("latency-mean", "l", 0, "artificial latency to inject (default: 0)")

	rootCmd.PersistentFlags().StringP("summary-path", "s", "/tmp/http_test_server_summary.json", "file to write out statistics summary to")
	rootCmd.PersistentFlags().StringP("parameters-path", "p", "", "file to write out test parameters to")

	rootCmd.PersistentFlags().UintP("rate-limit-bucket-capacity", "c", 0, "rate limit token bucket capacity (max tokens) (default: 0)")
	rootCmd.PersistentFlags().UintP("rate-limit-bucket-quantum", "q", 0, "rate limit token bucket quantum (tokens added per interval) (default: 0)")
	rootCmd.PersistentFlags().DurationP("rate-limit-bucket-fill-interval", "d", 0, "interval to refill quantum number of tokens (default: 0)")
	rootCmd.PersistentFlags().StringP("rate-limit-behavior", "b", "NONE", "behavior of rate limiter\nOne of [HARD, QUEUE, CLOSE, NONE].\nHARD returns 429s when limit is hit.\nQUEUE queues the request.\nCLOSE terminates the connection early\nNONE applies no limit.")
	rootCmd.PersistentFlags().Int("rate-limit-hard-status-code", http.StatusTooManyRequests, "status code to return for rate limit if behavior is HARD")

	viper.BindPFlags(rootCmd.PersistentFlags())

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("HTTP_TEST")
	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
