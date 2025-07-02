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
	fmt.Println("🔴 Test 1: REAL Redis Cluster Coordinator")
	
	redisCoord := scheduler.NewRealRedisClusterCoordinator("redis://localhost:6379")
	
	// Start coordinator
	err := redisCoord.Start(ctx)
	if err != nil {
		fmt.Printf("❌ Redis coordinator start failed: %v\n", err)
	} else {
		fmt.Printf("✅ Redis coordinator started\n")
		
		// Test leader election
		leaderResult := redisCoord.BecomeLeader(ctx)
		if leaderResult.IsSuccess() && leaderResult.Value() {
			fmt.Printf("✅ Became leader via Redis\n")
			
			// Check if still leader
			stillLeader := redisCoord.IsLeader(ctx)
			fmt.Printf("✅ Still leader: %v\n", stillLeader)
			
			// Test heartbeat
			err = redisCoord.SendHeartbeat(ctx)
			if err != nil {
				fmt.Printf("⚠️ Heartbeat failed: %v\n", err)
			} else {
				fmt.Printf("✅ Heartbeat sent\n")
			}
			
		} else if leaderResult.IsFailure() {
			fmt.Printf("⚠️ Leader election failed: %v\n", leaderResult.Error())
		} else {
			fmt.Printf("⚠️ Did not become leader (another leader exists)\n")
		}
		
		// Test cluster messaging
		err = redisCoord.SendMessage(ctx, "test-topic", map[string]interface{}{
			"type": "test",
			"data": "Hello from distributed test!",
		})
		if err != nil {
			fmt.Printf("⚠️ Message send failed: %v\n", err)
		} else {
			fmt.Printf("✅ Cluster message sent\n")
		}
		
		// Stop coordinator
		redisCoord.Stop()
		fmt.Printf("✅ Redis coordinator stopped\n")
	}

	// Test 2: etcd Coordinator  
	fmt.Println("\n🟢 Test 2: REAL etcd Coordinator")
	
	etcdCoord := scheduler.NewRealEtcdClusterCoordinator([]string{"localhost:2379"})
	
	// Start coordinator
	err = etcdCoord.Start(ctx)
	if err != nil {
		fmt.Printf("⚠️ etcd coordinator start failed (expected without etcd): %v\n", err)
	} else {
		fmt.Printf("✅ etcd coordinator started\n")
		
		// Test leader election
		leaderResult := etcdCoord.BecomeLeader(ctx)
		if leaderResult.IsSuccess() && leaderResult.Value() {
			fmt.Printf("✅ Became leader via etcd\n")
		} else if leaderResult.IsFailure() {
			fmt.Printf("⚠️ etcd leader election failed: %v\n", leaderResult.Error())
		}
		
		etcdCoord.Stop()
		fmt.Printf("✅ etcd coordinator stopped\n")
	}

	// Test 3: Network Coordinator
	fmt.Println("\n🌐 Test 3: REAL Network Coordinator")
	
	peers := []string{"localhost:8081", "localhost:8082"}
	networkCoord := scheduler.NewRealNetworkClusterCoordinator("localhost:8083", peers)
	
	// Start coordinator
	err = networkCoord.Start(ctx)
	if err != nil {
		fmt.Printf("⚠️ Network coordinator start failed: %v\n", err)
	} else {
		fmt.Printf("✅ Network coordinator started\n")
		
		// Test leader election
		leaderResult := networkCoord.BecomeLeader(ctx)
		if leaderResult.IsSuccess() && leaderResult.Value() {
			fmt.Printf("✅ Became leader via network\n")
		} else if leaderResult.IsFailure() {
			fmt.Printf("⚠️ Network leader election failed (expected without peers): %v\n", leaderResult.Error())
		}
		
		networkCoord.Stop()
		fmt.Printf("✅ Network coordinator stopped\n")
	}

	// Test 4: Timer Singleton with Redis
	fmt.Println("\n⏰ Test 4: REAL Timer Singleton")
	
	// Create new Redis coordinator for timer
	timerRedis := scheduler.NewRealRedisClusterCoordinator("redis://localhost:6379")
	err = timerRedis.Start(ctx)
	if err != nil {
		fmt.Printf("❌ Timer Redis start failed: %v\n", err)
	} else {
		fmt.Printf("✅ Timer Redis coordinator started\n")
		
		// Create timer singleton
		timerConfig := scheduler.TimerSingletonConfig{
			Name:        "test-timer",
			Interval:    3 * time.Second,
			Coordinator: timerRedis,
		}
		
		singleton := scheduler.NewTimerSingleton(timerConfig)
		
		// Set task function
		singleton.SetTask(func(ctx context.Context) error {
			fmt.Printf("✅ Timer singleton executed at %v\n", time.Now().Format("15:04:05"))
			return nil
		})
		
		// Start timer
		err = singleton.Start(ctx)
		if err != nil {
			fmt.Printf("❌ Timer singleton start failed: %v\n", err)
		} else {
			fmt.Printf("✅ Timer singleton started\n")
			
			// Let it run for a bit
			time.Sleep(5 * time.Second)
			
			// Stop timer
			singleton.Stop()
			fmt.Printf("✅ Timer singleton stopped\n")
		}
		
		timerRedis.Stop()
	}

	fmt.Println("\n🎯 DISTRIBUTED REAL TESTS COMPLETED")
}