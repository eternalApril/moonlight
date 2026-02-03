package testpipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestPipelining(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6380",
	})
	defer rdb.Close()

	ctx := context.Background()

	count := 10_000
	pipe := rdb.Pipeline()

	for i := 0; i < count; i++ {
		key := fmt.Sprintf("pipe_key_%d", i)
		val := fmt.Sprintf("val_%d", i)
		pipe.Set(ctx, key, val, 0)
	}

	getResults := make([]*redis.StringCmd, count)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("pipe_key_%d", i)
		getResults[i] = pipe.Get(ctx, key)
	}

	start := time.Now()
	_, err := pipe.Exec(ctx)
	elapsed := time.Since(start)

	assert.NoError(t, err, "Pipeline execution failed")
	fmt.Printf("Pipeline executed in %v\n", elapsed)

	for i := 0; i < count; i++ {
		expected := fmt.Sprintf("val_%d", i)
		val, err := getResults[i].Result()

		assert.NoError(t, err)
		assert.Equal(t, expected, val, "Key %d mismatch", i)
	}
}
