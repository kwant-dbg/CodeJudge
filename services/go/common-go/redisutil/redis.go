package redisutil

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func BuildOptions(redisURL string) (*redis.Options, error) {
	if !strings.Contains(redisURL, "://") {
		redisURL = "redis://" + redisURL
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	if opts.TLSConfig != nil {
		host := opts.Addr
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		opts.TLSConfig.MinVersion = tls.VersionTLS12
		if opts.TLSConfig.ServerName == "" {
			opts.TLSConfig.ServerName = host
		}
	}
	return opts, nil
}

func ConnectWithRetry(ctx context.Context, logger *zap.Logger, redisURL string, attempts int, delay time.Duration) *redis.Client {
	var err error
	for i := 0; i < attempts; i++ {
		opts, parseErr := BuildOptions(redisURL)
		if parseErr != nil {
			err = parseErr
			break
		}
		client := redis.NewClient(opts)
		if pingErr := client.Ping(ctx).Err(); pingErr == nil {
			logger.Info("Successfully connected to Redis")
			return client
		} else {
			err = pingErr
			client.Close()
		}
		time.Sleep(delay)
	}
	logger.Fatal("Could not connect to Redis", zap.Error(err))
	return nil
}
