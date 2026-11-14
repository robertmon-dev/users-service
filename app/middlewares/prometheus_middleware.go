package middlewares

import (
	"context"
	"fmt"
	"runtime"
	"strconv"

	"users-service/app/models/ent"
	"users-service/app/utils"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/cpu"
)

func (m *Middlewares) UpdateCacheHitRatioMetrics(redisClient *redis.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		cacheKey := ctx.Request.URL.Path
		_, err := redisClient.Get(ctx, cacheKey).Result()

		if err == redis.Nil {
			CacheMisses.Inc()
			ctx.Next()
		} else if err != nil {
			ctx.Next()
		} else {
			CacheHits.Inc()
			ctx.AbortWithStatusJSON(200, gin.H{})
		}
	}
}

func UpdateDBMetrics() ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			timer := prometheus.NewTimer(DBQueryDuration.WithLabelValues(mutation.Op().String(), mutation.Type()))
			defer timer.ObserveDuration()

			return next.Mutate(ctx, mutation)
		})
	}
}

func (m *Middlewares) UpdateServerMetrics() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ActiveRequests.Inc()
		defer ActiveRequests.Dec()

		ctx.Next()

		reqMethodAndPath := fmt.Sprintf("[%s] %s", ctx.Request.Method, ctx.FullPath())
		timer := prometheus.NewTimer(HttpRequestDuration.WithLabelValues(reqMethodAndPath))
		defer timer.ObserveDuration()

		responseSize := float64(ctx.Writer.Size())

		headerSize := 0
		for key, values := range ctx.Request.Header {
			headerSize += len(key)
			for _, value := range values {
				headerSize += len(value)
			}
		}

		bodySize := ctx.Request.ContentLength
		if bodySize < 0 {
			bodySize = 0
		}

		requestSize := float64(headerSize) + float64(bodySize)

		statusCode := strconv.Itoa(ctx.Writer.Status())

		if !utils.IsSuccessStatusCode(ctx.Writer.Status()) {
			ErrorRate.WithLabelValues(ctx.FullPath(), statusCode).Inc()
		} else {
			SuccessRate.WithLabelValues(ctx.FullPath(), statusCode).Inc()
		}

		HttpRequestCountWithPath.WithLabelValues(ctx.FullPath()).Inc()
		HttpRequestStatusCode.WithLabelValues(ctx.FullPath(), statusCode).Inc()
		HttpRequestSize.WithLabelValues(ctx.FullPath()).Observe(requestSize)
		HttpResponseSize.WithLabelValues(ctx.FullPath()).Observe(responseSize)
	}
}

func (m *Middlewares) UpdateSystemMetrics() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		percent, err := cpu.Percent(0, false)
		if err != nil {
			CPUUsage.Set(0)
		} else {
			CPUUsage.Set(percent[0])
		}

		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		MemoryUsage.Set(float64(memory.HeapAlloc) / 1024 / 1024)

		ThreadCount.Set(float64(runtime.NumGoroutine()))
		ctx.Next()
	}
}
