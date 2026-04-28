package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// StatsMiddleware counts requests for specific endpoint categories using Redis.
// Redis operations are fire-and-forget to avoid blocking the response.
func StatsMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() >= 400 {
			return
		}

		path := c.Request.URL.Path
		today := time.Now().Format("2006-01-02")
		ctx := context.Background()
		expireAt := time.Now().AddDate(0, 0, 1)

		var key string
		switch {
		case strings.HasPrefix(path, "/ai/"):
			key = fmt.Sprintf("stats:ai:%s", today)
		case strings.HasPrefix(path, "/search"):
			key = fmt.Sprintf("stats:search:%s", today)
		case strings.HasPrefix(path, "/files"):
			key = fmt.Sprintf("stats:file:%s", today)
		default:
			if strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "/health") && !strings.HasPrefix(path, "/metrics") {
				key = fmt.Sprintf("stats:api:%s", today)
			}
		}

		if key != "" {
			_ = rdb.Incr(ctx, key).Err()
			rdb.ExpireAt(ctx, key, expireAt)
		}

		if userID := c.GetString("user_id"); userID != "" {
			userKey := fmt.Sprintf("stats:active_users:%s", today)
			_ = rdb.SAdd(ctx, userKey, userID).Err()
			rdb.ExpireAt(ctx, userKey, expireAt)
		}
	}
}

// GetTodayCount returns the count for a given stats key prefix for today.
func GetTodayCount(ctx *gin.Context, rdb *redis.Client, prefix string) int64 {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("stats:%s:%s", prefix, today)
	val, _ := rdb.Get(ctx, key).Int64()
	return val
}

// GetActiveUsersCount returns the number of unique active users today.
func GetActiveUsersCount(ctx *gin.Context, rdb *redis.Client) int64 {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("stats:active_users:%s", today)
	count, _ := rdb.SCard(ctx, key).Result()
	return count
}
