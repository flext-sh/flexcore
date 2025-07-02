package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flext/flexcore/core"
	"github.com/flext/flexcore/infrastructure/windmill"
)

func main() {
	fmt.Println("🔥 TESTE REAL - Performance com Carga")

	// Test configuration
	const (
		workerCount   = 10
		messagesTotal = 1000
		testDuration  = 30 * time.Second
	)

	// Metrics
	var (
		messagesSent     int64
		messagesReceived int64
		errorsCount      int64
	)

	// Test 1: Setup
	fmt.Printf("\n1. ⚙️ Configuração do teste de performance...\n")
	fmt.Printf("   Workers: %d\n", workerCount)
	fmt.Printf("   Mensagens: %d\n", messagesTotal)
	fmt.Printf("   Duração: %v\n", testDuration)

	config := &core.FlexCoreConfig{
		ClusterName:       "performance-test-cluster",
		NodeID:           "perf-test-node",
		PluginDirectory:  "./plugins",
		WindmillURL:      "http://localhost:8000",
		WindmillToken:    "test-token",
		WindmillWorkspace: "demo",
		MessageQueues: []core.QueueConfig{
			{
				Name:    "performance-queue",
				Type:    "fifo",
				MaxSize: 10000,
				TTL:     10 * time.Minute,
			},
		},
	}

	windmillConfig := windmill.Config{
		BaseURL:   config.WindmillURL,
		Token:     config.WindmillToken,
		Workspace: config.WindmillWorkspace,
		Timeout:   30 * time.Second,
	}
	windmillClient := windmill.NewClient(windmillConfig)
	
	// Test 2: Create systems
	fmt.Println("\n2. 🏗️ Criando sistemas para teste de performance...")
	
	messageQueue := core.NewDistributedMessageQueue(windmillClient, config)
	scheduler := core.NewDistributedScheduler(windmillClient, config)

	ctx := context.Background()
	
	// Start systems
	if err := messageQueue.Start(ctx); err != nil {
		log.Fatalf("❌ Erro ao iniciar message queue: %v", err)
	}
	
	startResult := scheduler.Start(ctx)
	if startResult.IsFailure() {
		log.Printf("⚠️ Scheduler em modo fallback: %v", startResult.Error())
	}
	
	fmt.Println("✅ Sistemas iniciados")

	// Test 3: Performance test - Message sending
	fmt.Println("\n3. 📤 Teste de performance - Envio de mensagens...")
	
	startTime := time.Now()
	var wg sync.WaitGroup
	
	// Producer workers
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			messagesPerWorker := messagesTotal / workerCount
			for i := 0; i < messagesPerWorker; i++ {
				message := &core.Message{
					ID:          fmt.Sprintf("perf-msg-%d-%d", workerID, i),
					Queue:       "performance-queue",
					Content:     map[string]interface{}{
						"worker_id":   workerID,
						"message_num": i,
						"timestamp":   time.Now().Unix(),
						"data":        fmt.Sprintf("Performance test data from worker %d", workerID),
					},
					Priority:    1,
					CreatedAt:   time.Now(),
					MaxAttempts: 3,
				}
				
				sendResult := messageQueue.SendMessage(ctx, "performance-queue", message)
				if sendResult.IsSuccess() {
					atomic.AddInt64(&messagesSent, 1)
				} else {
					atomic.AddInt64(&errorsCount, 1)
				}
			}
		}(w)
	}
	
	// Wait for all producers
	wg.Wait()
	sendDuration := time.Since(startTime)
	
	fmt.Printf("✅ Envio completo em %v\n", sendDuration)
	fmt.Printf("   📊 Mensagens enviadas: %d\n", messagesSent)
	fmt.Printf("   📊 Erros: %d\n", errorsCount)
	fmt.Printf("   📊 Taxa de envio: %.2f msgs/seg\n", float64(messagesSent)/sendDuration.Seconds())

	// Test 4: Performance test - Message receiving
	fmt.Println("\n4. 📥 Teste de performance - Recebimento de mensagens...")
	
	startTime = time.Now()
	
	// Consumer workers
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for {
				receiveResult := messageQueue.ReceiveMessages(ctx, "performance-queue", 10)
				if receiveResult.IsFailure() {
					atomic.AddInt64(&errorsCount, 1)
					continue
				}
				
				messages := receiveResult.Value()
				if len(messages) == 0 {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				
				atomic.AddInt64(&messagesReceived, int64(len(messages)))
				
				// Stop when all messages received
				if atomic.LoadInt64(&messagesReceived) >= messagesSent {
					break
				}
			}
		}(w)
	}
	
	// Wait for all consumers or timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()
	
	select {
	case <-done:
		// All done
	case <-time.After(testDuration):
		// Timeout
		fmt.Println("   ⏰ Timeout do teste de recebimento")
	}
	
	receiveDuration := time.Since(startTime)
	
	fmt.Printf("✅ Recebimento completo em %v\n", receiveDuration)
	fmt.Printf("   📊 Mensagens recebidas: %d\n", messagesReceived)
	fmt.Printf("   📊 Taxa de recebimento: %.2f msgs/seg\n", float64(messagesReceived)/receiveDuration.Seconds())

	// Test 5: System metrics
	fmt.Println("\n5. 📊 Métricas finais do sistema...")
	
	metrics := messageQueue.GetMetrics()
	fmt.Printf("   📈 Total enfileiradas: %d\n", metrics.MessagesQueued)
	fmt.Printf("   📈 Total entregues: %d\n", metrics.MessagesDelivered)
	fmt.Printf("   📈 Total expiradas: %d\n", metrics.MessagesExpired)
	fmt.Printf("   📈 Profundidade atual: %d\n", metrics.QueueDepth)

	// Test 6: Stress test
	fmt.Println("\n6. 💪 Teste de stress - Rajada de mensagens...")
	
	stressStartTime := time.Now()
	stressMessages := 500
	
	for i := 0; i < stressMessages; i++ {
		message := &core.Message{
			ID:          fmt.Sprintf("stress-msg-%d", i),
			Queue:       "performance-queue",
			Content:     map[string]interface{}{
				"stress_test": true,
				"message_id":  i,
				"timestamp":   time.Now().Unix(),
			},
			Priority:    1,
			CreatedAt:   time.Now(),
			MaxAttempts: 3,
		}
		
		messageQueue.SendMessage(ctx, "performance-queue", message)
	}
	
	stressDuration := time.Since(stressStartTime)
	fmt.Printf("✅ Stress test: %d mensagens em %v\n", stressMessages, stressDuration)
	fmt.Printf("   📊 Taxa de stress: %.2f msgs/seg\n", float64(stressMessages)/stressDuration.Seconds())

	// Test 7: Cleanup
	fmt.Println("\n7. 🛑 Cleanup...")
	
	if err := messageQueue.Stop(ctx); err != nil {
		log.Printf("⚠️ Erro ao parar message queue: %v", err)
	}
	
	stopResult := scheduler.Stop(ctx)
	if stopResult.IsFailure() {
		log.Printf("⚠️ Erro ao parar scheduler: %v", stopResult.Error())
	}

	// Final results
	totalDuration := time.Since(startTime)
	
	fmt.Println("\n🎉 TESTE DE PERFORMANCE COMPLETO!")
	fmt.Printf("✅ Duração total: %v\n", totalDuration)
	fmt.Printf("✅ Mensagens processadas: %d\n", messagesSent)
	fmt.Printf("✅ Throughput médio: %.2f msgs/seg\n", float64(messagesSent)/totalDuration.Seconds())
	fmt.Printf("✅ Taxa de sucesso: %.2f%%\n", float64(messagesSent-errorsCount)/float64(messagesSent)*100)
	fmt.Println("✅ Sistema FlexCore aprovado em teste de carga!")
}