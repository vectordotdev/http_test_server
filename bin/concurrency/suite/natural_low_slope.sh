HTTP_TEST_DESCRIPTION="Natural rate limit, low load slope"
HTTP_TEST_LATENCY_DISTRIBUTION="EXPRESSION"
HTTP_TEST_LATENCY_EXPRESSION_MEAN_MS="200 + active_requests * 5 + (0<=active_requests && active_requests<10 ? 0 : 2 ** (active_requests - 10))"
HTTP_TEST_EXPECTED_RATE=30
