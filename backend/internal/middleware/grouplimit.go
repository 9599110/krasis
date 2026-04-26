package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krasis/krasis/pkg/response"
	"github.com/redis/go-redis/v9"
)

// GroupLimitMiddleware checks user's group feature limits (e.g. ai_ask_limit).
// Feature config format in group_features: {"value": N, "period": "minute"|"hour"|"day"}
// Group/feature config is cached in Redis for 5 minutes to avoid repeated DB queries.
func GroupLimitMiddleware(pool *pgxpool.Pool, rdb *redis.Client, featureKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetString("user_id")
		if userIDStr == "" {
			c.Next()
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.Next()
			return
		}

		// Try cached config first
		cacheKey := fmt.Sprintf("limit_config:%s:%s", featureKey, userIDStr)
		var fv struct {
			Value  int    `json:"value"`
			Period string `json:"period"`
		}

		cached, err := rdb.Get(c.Request.Context(), cacheKey).Result()
		if err == nil && cached != "" {
			if json.Unmarshal([]byte(cached), &fv) == nil && fv.Value > 0 {
				// Cached config is valid, proceed with limit check
			} else {
				// Stale config, load from DB
				fv = loadFeatureConfig(c.Request.Context(), pool, userID, featureKey)
				if fv.Value > 0 {
					b, _ := json.Marshal(fv)
					rdb.Set(c.Request.Context(), cacheKey, string(b), 5*time.Minute)
				}
			}
		} else {
			// Not cached or Redis error, load from DB
			fv = loadFeatureConfig(c.Request.Context(), pool, userID, featureKey)
			if fv.Value > 0 {
				b, _ := json.Marshal(fv)
				rdb.Set(c.Request.Context(), cacheKey, string(b), 5*time.Minute)
			}
		}

		if fv.Value <= 0 {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", featureKey, userIDStr)
		count, err := rdb.Get(c.Request.Context(), key).Int64()
		if err != nil && err != redis.Nil {
			c.Next()
			return
		}

		if count >= int64(fv.Value) {
			response.Error(c, 429, response.ErrTooManyRequests, "请求频率超出限制，请稍后再试")
			c.Abort()
			return
		}

		pipe := rdb.Pipeline()
		pipe.Incr(c.Request.Context(), key)
		pipe.Expire(c.Request.Context(), key, time.Duration(periodSeconds(fv.Period))*time.Second)
		_, _ = pipe.Exec(c.Request.Context())

		c.Next()
	}
}

func loadFeatureConfig(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, featureKey string) (fv struct {
	Value  int    `json:"value"`
	Period string `json:"period"`
}) {
	var groupID uuid.UUID
	if err := pool.QueryRow(ctx,
		"SELECT group_id FROM users WHERE id = $1", userID,
	).Scan(&groupID); err != nil || groupID == uuid.Nil {
		return
	}

	var featureVal []byte
	if err := pool.QueryRow(ctx,
		"SELECT feature_value FROM group_features WHERE group_id = $1 AND feature_key = $2",
		groupID, featureKey,
	).Scan(&featureVal); err != nil {
		return
	}

	_ = json.Unmarshal(featureVal, &fv)
	return
}

func periodSeconds(period string) int64 {
	switch period {
	case "minute":
		return 60
	case "hour":
		return 3600
	case "day":
		return 86400
	default:
		return 60
	}
}
