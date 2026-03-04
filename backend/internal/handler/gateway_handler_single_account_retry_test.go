package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// sleepWithContext 测试（替代原 sleepAntigravitySingleAccountBackoff）
// ---------------------------------------------------------------------------

func TestSleepWithContext_ReturnsTrue(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	ok := sleepWithContext(ctx, singleAccountBackoffDelay)
	elapsed := time.Since(start)

	require.True(t, ok, "should return true when context is not canceled")
	// 固定延迟 2s
	require.GreaterOrEqual(t, elapsed, 1500*time.Millisecond, "should wait approximately 2s")
	require.Less(t, elapsed, 5*time.Second, "should not wait too long")
}

func TestSleepWithContext_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	start := time.Now()
	ok := sleepWithContext(ctx, singleAccountBackoffDelay)
	elapsed := time.Since(start)

	require.False(t, ok, "should return false when context is canceled")
	require.Less(t, elapsed, 500*time.Millisecond, "should return immediately on cancel")
}

func TestSleepWithContext_ZeroDuration(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	ok := sleepWithContext(ctx, 0)
	elapsed := time.Since(start)

	require.True(t, ok, "should return true for zero duration")
	require.Less(t, elapsed, 100*time.Millisecond, "should return immediately for zero duration")
}
