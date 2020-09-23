HTTP_TEST_DESCRIPTION="Natural rate limit, load removed mid-test"
HTTP_TEST_LATENCY_DISTRIBUTION="EXPRESSION"
HTTP_TEST_LATENCY_EXPRESSION_MEAN_MS="200 + 10 * (sin((1/60) * 2 * pi * t) < 0 ? (active_requests<5 ? active_requests : (5-1) + 2 ** (active_requests - 5)) : 2 ** (active_requests - 1))"
