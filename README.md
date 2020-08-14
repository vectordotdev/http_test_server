Automated # HTTP Test Server

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
$ HTTP_TEST_LATENCY_MEAN=200ms TEST_TIME=60 ./bin/run-concurrency-test
```

This will run a 60s test of vector using an artificial latency of 200ms.

It will output the directory to which it will write the test artifacts
(controllable via `$OUTPUT_DIR`).

The artifacts will include:

* `concurrent_requests.dat`: a gnuplot dat file with the concurrent requests at
  every 100ms interval
* `concurrent_requests.svg`: a plot of the concurrent requests at each 100ms
* `parameters.json`: the test paramaters
* `summary.json`: the test result including each request

Additionally, you will find output files for the run processes that can be
useful for debugging or further understanding behavior during the test:

* `server.log`: HTTP test server stdout
* `server.err`: HTTP test server stderr
* `vector.log` Vector stdout
* `vector.err`: Vector stderr

Available environment variables:

* `OUTPUT_DIR`: where to write the test artifacts (defaults to a tmpdir)
* `VECTOR`: the path to the `vector` binary (defaults to `vector`)
* `HTTP_TEST_LATENCY_MEAN`: artificial latency
* `HTTP_TEST_RATE_LIMIT_BEHAVIOR`: the behavior of the rate limiting. Possible
  values: `NONE` (no rate limit; the default), `HARD` (return a HTTP 429 when
  limit is hit) and `QUEUE` (queue the request until there is available
  capacity).
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
