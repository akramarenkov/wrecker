package limiter

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	positiveIterationsExcess = 3
)

func TestLimiter(t *testing.T) {
	for limit := range 1 << 8 {
		testLimiterCalls(t, limit)
	}

	for limit := range 1 << 15 {
		testLimiterSize(t, 1024, limit)
	}

	for divider := range 1 << 8 {
		testLimiterSize(t, math.MaxInt/(divider+1), math.MaxInt)
	}

	for subtrahend := range 1 << 8 {
		testLimiterSize(t, math.MaxInt-subtrahend, math.MaxInt)
	}
}

func testLimiterCalls(t *testing.T, callsLimit int) {
	negativeIterations := callsLimit
	require.GreaterOrEqual(t, negativeIterations, 0, "calls limit: %v", callsLimit)

	positiveIterations := positiveIterationsExcess * (negativeIterations + 1)
	require.Positive(t, positiveIterations, "calls limit: %v", callsLimit)

	limiter := New(callsLimit, -1)

	for range negativeIterations {
		require.False(t, limiter.IsReached(0), "calls limit: %v", callsLimit)
	}

	for range positiveIterations {
		require.True(t, limiter.IsReached(0), "calls limit: %v", callsLimit)
	}
}

func testLimiterSize(t *testing.T, blockSize int, sizeLimit int) {
	negativeIterations := sizeLimit / blockSize
	require.GreaterOrEqual(
		t,
		negativeIterations,
		0,
		"block size: %v, size limit: %v",
		blockSize,
		sizeLimit,
	)

	positiveIterations := positiveIterationsExcess * (negativeIterations + 1)
	require.Positive(
		t,
		positiveIterations,
		"block size: %v, size limit: %v",
		blockSize,
		sizeLimit,
	)

	limiter := New(-1, sizeLimit)

	for range negativeIterations {
		require.False(
			t,
			limiter.IsReached(blockSize),
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)
	}

	for range positiveIterations {
		require.True(
			t,
			limiter.IsReached(blockSize),
			"block size: %v, size limit: %v",
			blockSize,
			sizeLimit,
		)
	}
}
