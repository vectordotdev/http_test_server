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
	LatencyDistribution                        string  `json:"latency_distribution"`
	LatencyDistributionNormalMean              *string `json:"latency_distribution_normal_mean,omitempty"`
	LatencyDistributionNormalStandardDeviation *string `json:"latency_distribution_normal_standard_deviation,omitempty"`

	LatencyDistributionExpressionMean              *string `json:"latency_distribution_expression_mean,omitempty"`
	LatencyDistributionExpressionStandardDeviation *string `json:"latency_distribution_expression_standard_deviation,omitempty"`

	RateLimitBehavior           string  `json:"rate_limit_behavior"`
	RateLimitBucketFillInterval *string `json:"rate_limit_bucket_fill_interval,omitempty"`
	RateLimitBucketCapacity     *int64  `json:"rate_limit_bucket_capaticy,omitempty"`
	RateLimitBucketQuauntum     *int64  `json:"rate_limit_bucket_quantum,omitempty"`
	RateLimitHardStatusCode     *int    `json:"rate_limit_hard_status_code,omitempty"`
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

		latencyDistribution := viper.GetString("latency-distribution")
		parameters.LatencyDistribution = latencyDistribution
		switch latencyDistribution {
		case "NORMAL":
			mean := viper.GetDuration("latency-normal-mean")
			stddev := viper.GetDuration("latency-normal-stddev")
			parameters.LatencyDistributionNormalMean = func() *string {
				s := mean.String()
				return &s
			}()
			parameters.LatencyDistributionNormalStandardDeviation = func() *string {
				s := stddev.String()
				return &s
			}()
			opts = append(opts, WithLatency(NewLatencyMiddlewareNormal(mean, stddev)))
		case "EXPRESSION":
			mean := viper.GetString("latency-expression-mean-ms")
			parameters.LatencyDistributionExpressionMean = &mean

			stddev := viper.GetString("latency-expression-stddev-ms")
			parameters.LatencyDistributionExpressionStandardDeviation = &stddev

			middleware, err := NewLatencyMiddlewareExpression(mean, stddev)
			if err != nil {
				return fmt.Errorf("latency expression error: %s", err)
			}

			opts = append(opts, WithLatency(middleware))
		default:
			return fmt.Errorf("unknown latency-distribution value: %s", latencyDistribution)
		}

		behavior := viper.GetString("rate-limit-behavior")
		if behavior != "NONE" {
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

			parameters.RateLimitBucketFillInterval = func() *string {
				s := fillInterval.String()
				return &s
			}()
			parameters.RateLimitBucketCapacity = &capacity
			parameters.RateLimitBucketQuauntum = &quantum

			opts = append(opts, WithRateLimiter(rateLimiter))
		}

		parameters.RateLimitBehavior = behavior

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

	rootCmd.PersistentFlags().StringP("latency-distribution", "l", "NORMAL", "distribution of artificial latency\nOne of [NORMAL,FUNCTION]")
	rootCmd.PersistentFlags().DurationP("latency-normal-mean", "m", 0, "artificial latency to inject; only applies when latency-distribution is NORMAL (default: 0)")
	rootCmd.PersistentFlags().DurationP("latency-normal-stddev", "S", 0, "standard deviation of artificial latency to inject; only applies when latency-distribution is NORMAL (default: 0)")

	rootCmd.PersistentFlags().String("latency-expression-mean-ms", "0", "expression to use to evaluate latency of request in ms; variables: [concurrent_requests]; only applies when latency-distribution is EXPRESSION (default: '0')")
	rootCmd.PersistentFlags().String("latency-expression-stddev-ms", "0", "expression to use to evaluate stddev of the latency of request in ms; variables: [concurrent_requests]; only applies when latency-distribution is EXPRESSION (default: '0')")

	rootCmd.PersistentFlags().StringP("summary-path", "s", "/tmp/http_test_server_summary.json", "file to write out statistics summary to")
	rootCmd.PersistentFlags().StringP("parameters-path", "p", "", "file to write out test parameters to")

	rootCmd.PersistentFlags().UintP("rate-limit-bucket-capacity", "c", 0, "rate limit token bucket capacity (max tokens) (default: 0)")
	rootCmd.PersistentFlags().UintP("rate-limit-bucket-quantum", "q", 0, "rate limit token bucket quantum (tokens added per interval) (default: 0)")
	rootCmd.PersistentFlags().DurationP("rate-limit-bucket-fill-interval", "d", 0, "interval to refill quantum number of tokens (default: 0)")
	rootCmd.PersistentFlags().StringP("rate-limit-behavior", "b", "NONE", "behavior of rate limiter\nOne of [HARD, QUEUE, CLOSE, NONE].\nHARD returns 429s when limit is hit.\nQUEUE queues the request.\nCLOSE terminates the connection early\nNONE applies no limit.")
	rootCmd.PersistentFlags().Int("rate-limit-hard-status-code", http.StatusTooManyRequests, "status code to return for rate limit; only applies if rate-limit-behavior is HARD")

	viper.BindPFlags(rootCmd.PersistentFlags())

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("HTTP_TEST")
	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
