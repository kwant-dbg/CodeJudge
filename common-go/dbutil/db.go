package dbutil

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func ConnectWithRetry(logger *zap.Logger, databaseURL string, attempts int, delay time.Duration) *sql.DB {
	var err error
	for i := 0; i < attempts; i++ {
		conn, openErr := sql.Open("postgres", databaseURL)
		if openErr != nil {
			err = openErr
		} else {
			if pingErr := conn.Ping(); pingErr == nil {
				logger.Info("Successfully connected to the database")
				return conn
			} else {
				err = pingErr
				conn.Close()
			}
		}
		time.Sleep(delay)
	}
	logger.Fatal("Failed to connect to the database", zap.Error(err))
	return nil
}

func ReadyCheck(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}
