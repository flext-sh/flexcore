package main

import (
	"context"
	"fmt"
	"time"

	"github.com/flext/flexcore/infrastructure/scheduler"
)

func main() {
	fmt.Println("🌐 Testing REAL Distributed Coordination...")

	ctx := context.Background()

	// Test 1: Redis Coordinator
	fmt.Println("🔴 Test 1: REAL Redis Coordinator")
	
	redisCoord, err := scheduler.NewRedisCoordinator("localhost:6379", "")
	if err != nil {
		fmt.Printf("❌ Redis coordinator creation failed: %v\n", err)
	} else {
		// Test distributed locking
		lockKey := "test-lock"
		nodeID := "test-node-1"
		ttl := 10 * time.Second

		acquired, err := redisCoord.AcquireLock(ctx, lockKey, nodeID, ttl)
		if err != nil {
			fmt.Printf("❌ Redis lock acquire failed: %v\n", err)
		} else if acquired {
			fmt.Printf("✅ Redis lock acquired by %s\n", nodeID)
			
			// Test lock renewal
			err = redisCoord.RenewLock(ctx, lockKey, nodeID, ttl)
			if err != nil {
				fmt.Printf("⚠️ Redis lock renewal failed: %v\n", err)
			} else {
				fmt.Printf("✅ Redis lock renewed\n")
			}
			
			// Release lock
			err = redisCoord.ReleaseLock(ctx, lockKey, nodeID)
			if err != nil {
				fmt.Printf("❌ Redis lock release failed: %v\n", err)
			} else {
				fmt.Printf("✅ Redis lock released\n")
			}
		} else {
			fmt.Printf("❌ Redis lock not acquired\n")
		}

		// Test leader election
		leaderKey := "test-leader"
		isLeader, err := redisCoord.BecomeLeader(ctx, leaderKey, nodeID, ttl)
		if err != nil {
			fmt.Printf("❌ Redis leader election failed: %v\n", err)
		} else if isLeader {
			fmt.Printf("✅ Became leader via Redis\n")
			
			// Check if still leader
			stillLeader := redisCoord.IsLeader(ctx, leaderKey, nodeID)
			fmt.Printf("✅ Still leader: %v\n", stillLeader)
			
			// Resign leadership
			err = redisCoord.ResignLeadership(ctx, leaderKey, nodeID)
			if err != nil {
				fmt.Printf("❌ Redis leadership resignation failed: %v\n", err)
			} else {
				fmt.Printf("✅ Leadership resigned\n")
			}
		}
	}

	// Test 2: Network Coordinator  
	fmt.Println("\n🌐 Test 2: REAL Network Coordinator")
	
	peers := []string{"localhost:8081", "localhost:8082"}
	networkCoord := scheduler.NewNetworkCoordinator("localhost:8083", peers)
	
	// Test basic functionality
	err = networkCoord.Start(ctx)
	if err != nil {
		fmt.Printf("❌ Network coordinator start failed: %v\n", err)
	} else {
		fmt.Printf("✅ Network coordinator started\n")
		
		// Test leader election (will fail without real peers, but code path works)
		leaderKey := "net-leader"
		nodeID := "net-node-1"
		isLeader, err := networkCoord.BecomeLeader(ctx, leaderKey, nodeID, time.Minute)
		if err != nil {
			fmt.Printf("⚠️ Network leader election failed (expected without peers): %v\n", err)
		} else {
			fmt.Printf("✅ Network leader election: %v\n", isLeader)
		}
		
		networkCoord.Stop()
		fmt.Printf("✅ Network coordinator stopped\n")
	}

	// Test 3: Timer Singleton
	fmt.Println("\n⏰ Test 3: REAL Timer Singleton")
	
	// Create timer singleton with Redis coordination
	singleton := scheduler.NewTimerSingleton("test-timer", 5*time.Second, redisCoord)
	
	// Start timer
	err = singleton.Start(ctx, func(ctx context.Context) error {
		fmt.Printf("✅ Timer singleton executed at %v\n", time.Now().Format("15:04:05"))
		return nil
	})
	
	if err != nil {
		fmt.Printf("❌ Timer singleton start failed: %v\n", err)
	} else {
		fmt.Printf("✅ Timer singleton started\n")
		
		// Let it run for a bit
		time.Sleep(3 * time.Second)
		
		// Stop timer
		singleton.Stop()
		fmt.Printf("✅ Timer singleton stopped\n")
	}

	fmt.Println("\n🎯 DISTRIBUTED REAL TESTS COMPLETED")
}