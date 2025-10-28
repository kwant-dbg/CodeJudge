package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const (
	redisAddr      = "localhost:6379"
	queueName      = "submission_queue"
	numJobsToPush = 1000
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	ctx := context.Background()

	// Assume submissions with IDs 1 to 1000 exist in the database
	log.Printf("Pushing %d jobs to Redis queue '%s'...\n", numJobsToPush, queueName)
	for i := 1; i <= numJobsToPush; i++ {
		err := rdb.LPush(ctx, queueName, strconv.Itoa(i)).Err()
		if err != nil {
			log.Fatalf("Failed to push job %d to Redis: %v", i, err)
		}
	}
	log.Printf("Successfully pushed %d jobs.\n", numJobsToPush)
}
