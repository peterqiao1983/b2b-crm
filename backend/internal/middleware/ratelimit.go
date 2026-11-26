package middleware

import (
	"net/http"
	"sync"
	"time"

	"b2b-crm/internal/response"

	"github.com/gin-gonic/gin"
)

func LoginRateLimit(maxAttempts int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := make(map[string][]time.Time)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		cutoff := now.Add(-window)

		mu.Lock()
		times := buckets[ip]
		kept := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) >= maxAttempts {
			buckets[ip] = kept
			mu.Unlock()
			response.Error(c, http.StatusTooManyRequests, "登录尝试过于频繁，请 1 分钟后再试")
			c.Abort()
			return
		}
		buckets[ip] = append(kept, now)
		if len(buckets) > 2000 {
			for k, arr := range buckets {
				if len(arr) == 0 || !arr[len(arr)-1].After(cutoff) {
					delete(buckets, k)
				}
			}
		}
		mu.Unlock()
		c.Next()
	}
}
