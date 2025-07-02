package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Connect to REAL Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()

	// Test 1: Basic connection
	fmt.Println("🔴 Testing REAL Redis connection...")
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		fmt.Printf("❌ Redis connection failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Redis PONG: %s\n", pong)

	// Test 2: Distributed locking (REAL)
	fmt.Println("\n🔒 Testing REAL distributed locking...")
	nodeID := "test-node-1"
	lockKey := "flx:lock:test"
	ttl := 10 * time.Second

	// Try to acquire lock
	acquired, err := rdb.SetNX(ctx, lockKey, nodeID, ttl).Result()
	if err != nil {
		fmt.Printf("❌ Lock acquire failed: %v\n", err)
		return
	}

	if acquired {
		fmt.Printf("✅ Lock acquired by %s\n", nodeID)
		
		// Try to acquire same lock with different node (should fail)
		acquired2, err := rdb.SetNX(ctx, lockKey, "test-node-2", ttl).Result()
		if err != nil {
			fmt.Printf("❌ Second lock test failed: %v\n", err)
			return
		}
		
		if !acquired2 {
			fmt.Printf("✅ Second node correctly denied lock\n")
		} else {
			fmt.Printf("❌ Lock not working - second node got lock\n")
		}
		
		// Release lock
		rdb.Del(ctx, lockKey)
		fmt.Printf("✅ Lock released\n")
	} else {
		fmt.Printf("❌ Could not acquire lock\n")
	}

	// Test 3: Pub/Sub (REAL)
	fmt.Println("\n📡 Testing REAL pub/sub...")
	pubsub := rdb.Subscribe(ctx, "flx:events:test")
	defer pubsub.Close()

	// Publish message
	err = rdb.Publish(ctx, "flx:events:test", "Hello from FlexCore").Err()
	if err != nil {
		fmt.Printf("❌ Publish failed: %v\n", err)
		return
	}

	// Receive message
	msg, err := pubsub.ReceiveTimeout(ctx, 2*time.Second)
	if err != nil {
		fmt.Printf("❌ Receive failed: %v\n", err)
		return
	}

	switch m := msg.(type) {
	case *redis.Message:
		fmt.Printf("✅ Received message: %s\n", m.Payload)
	default:
		fmt.Printf("❌ Unexpected message type\n")
	}

	fmt.Println("\n🎯 REDIS REAL TESTS COMPLETED")
}