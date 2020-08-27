# Automated HTTP Test Server

A simple HTTP server useful for testing.

This is used in [Vector]'s [test harness] to test and benchmark HTTP performance.

## Getting started

1. Run `go build`
2. Run `./http_test_server`

## Testing

The HTTP test server exposes some configuration that can be used to artifically
induce request behavior including:

* rate limiting
* latency

You can run `./http_test_server -h` to see all of the options.

## Vector Concurrency Testing

Additionally, there are some executables in `./bin` that allow for running
a concurrency test using vector and analyzing the results.

To run a single test, you can run:

```bash
$ HTTP_TEST_LATENCY_NORMAL_MEAN=200ms HTTP_TEST_LATENCY_NORMAL_STDDEV=50ms TEST_TIME=60 ./bin/run-concurrency-test
```

This will run a 60s test of vector using an artificial latency of a mean of
200ms and standard deviation of 50ms.

It will output the directory to which it will write the test artifacts
(controllable via `$OUTPUT_DIR`).

The artifacts will include:

* `concurrent_requests.dat`: a gnuplot dat file with the concurrent requests at
  every 100ms interval
* `requests.dat`: a gnuplot dat file with the request intervals
* `request_rate.dat`: a gnuplot dat file with per second request rate
* `plot.svg`: a plot of the test output
* `plot.png`: the same plot as a png to ease uploading to Github
  which doesn't support svgs
* `parameters.json`: the test paramaters
* `summary.json`: the test result including each request

Additionally, you will find output files for the run processes that can be
useful for debugging or further understanding behavior during the test:

* `server.log`: HTTP test server stdout
* `server.err`: HTTP test server stderr
* `test_cmd.log` Test command (typically vector) stdout
* `test_cmd.err`: Vector (typically vector) stderr

Available environment variables:

* `OUTPUT_DIR`: where to write the test artifacts (defaults to a tmpdir)
* `VECTOR`: the path to the `vector` binary (defaults to `vector`)
* `TEST_CMD`: the test command to execute. It can use the `URL` environment
  variable to configure the command. Defaults to running `$VECTOR` but can be
  used to run other tools like `ab` (e.g. `TEST_CMD='ab -t ${TEST_TIME} -n 10000
  -c 100 -m POST ${URL}'`)
* `HTTP_TEST_LATENCY_DISTRIBUTION`: artificial latency distribution. One of:
  `NORMAL` for latencies distributed using the normal distribution; `EXPRESSION`
  to provide an expression to calculate the mean / stddev depending on other
  parameters (see below for expression details)
  `NORMAL` is currently supported (the default)
* `HTTP_TEST_LATENCY_NORMAL_MEAN`: artificial latency mean for the `NORMAL`
  distribution
* `HTTP_TEST_LATENCY_NORMAL_STDDEV`: artificial latency standard devation for
  the `NORMAL` distribution
* `HTTP_TEST_LATENCY_EXPRESSION_MEAN_MS`: an expression to calcelate the mean
  latency of the request. See below for expression details.
* `HTTP_TEST_LATENCY_EXPRESSION_STDDEV_MS`: an expression to calcelate the
  stddev of the request latency. See below for expression details.
* `HTTP_TEST_ERROR_EXPRESSION`: expression to evaluate to determine if the request should error. See below for expression details. It is expected to return one of: false if the request should not error; true if the request should error with 500; an integer value if the request should error with the given HTTP status code; the string 'CLOSE' if the request should error by simply closing the connection
  latency of the request. See below for expression details.
* `HTTP_TEST_RATE_LIMIT_BEHAVIOR`: the behavior of the rate limiting. Possible
  values: `NONE` (no rate limit; the default); `HARD` (return a HTTP 429 when
  limit is hit); `CLOSE` (close the connection without response when limit is
  hit); and `QUEUE` (queue the request until there is available capacity).
* `HTTP_TEST_RATE_LIMIT_HARD_STATUS_CODE`: the status code to return if
  `HTTP_TEST_RATE_LIMIT_BEHAVIOR` is `HARD` (defaults to 429)
* `HTTP_TEST_RATE_LIMIT_BUCKET_CAPACITY`: The maximum number of rate limit
  tokens
* `HTTP_TEST_RATE_LIMIT_BUCKET_QUANTUM`: the number of tokens to add per fill
  interval
* `HTTP_TEST_RATE_LIMIT_BUCKET_FILL_INTERVAL`: the fill interval to add quantum
  tokens

IF `HTTP_TEST_RATE_LIMIT_BEHAVIOR` is not set to `NONE` all of the other
`HTTP_TEST_RATE_LIMIT_*` variables must be set.

Example:

```bash
HTTP_TEST_LATENCY_MEAN=500ms \
HTTP_TEST_RATE_LIMIT_BUCKET_FILL_INTERVAL=1s \
HTTP_TEST_RATE_LIMIT_BUCKET_CAPACITY=5 \
HTTP_TEST_RATE_LIMIT_BUCKET_QUANTUM=5 \
HTTP_TEST_RATE_LIMIT_BEHAVIOR=HARD \
./http_test_server
```

This will run the test server with a simulated latency of 500ms and a hard rate
limit of 5 requests per second (refreshed every second).

#### Expression support

When using `HTTP_TEST_LATENCY_DISTRIBUTION=EXPRESSION` an expression can be
provided to calculate the latency based on other variables. This is useful, for
example, to increase the latency based on the number of active requests.

The expressions are expected to resolve to a number of ms.

For example, providing `active_requests ^ 2` for the mean would cause the mean
latency for that request to be the number of in-flight requests, squared.

When using `HTTP_TEST_ERROR_EXPRESSION` an expression can be provided to
determine if the active request should be allowed through or errored.

For all expression, supported variables are:

* `active_requests`: the number of currently active requests (including this
  one)

For all expression, supported functioare:

* `rand()`: return a random number in [0.0,1.0)

Complete operator support can be found in the [govaluate
documentation](https://github.com/Knetic/govaluate/blob/master/MANUAL.md#operators).
This library is used to evaluate the expressions.

### Running the concurrency test suite

There is a suite of concurrency tests using various parameters defined in
`./bin/concurrency/suite.json`. Each test is simply a set of environment
variables that is set for the test.

A `HTTP_TEST_NAME` is required and indicates the name of the test (used as the
result directory).

This suite can be run via:

```bash
./bin/run-concurrency-test-suite
```

You shouldn't typically need to change the configuration, but there are some
environment variables that can be used to modify the behavior (see script).

It will output the directory which will contain a subdirectory for each test
with the test artifacts.

[test harness]: https://github.com/timberio/vector-test-harness
[Vector]: https://github.com/timberio/vector
