HTTP_TEST_DESCRIPTION="Natural rate limit, low load slope"
HTTP_TEST_LATENCY_DISTRIBUTION="EXPRESSION"
HTTP_TEST_LATENCY_EXPRESSION_MEAN_MS="200 + active_requests * 10 + (active_requests<10 ? 0 : 2 ** (active_requests - 10))"
