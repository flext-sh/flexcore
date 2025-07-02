// test_stress_prolonged_10min.go - REAL 10+ minute stress test
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

type StressTestMetrics struct {
	TestStartedAt        int64   `json:"test_started_at"`
	TestCompletedAt      int64   `json:"test_completed_at"`
	TestDurationSeconds  int64   `json:"test_duration_seconds"`
	TotalMessagesSent    int64   `json:"total_messages_sent"`
	TotalMessagesReceived int64  `json:"total_messages_received"`
	AverageMessagesPerSec float64 `json:"average_messages_per_sec"`
	PeakMessagesPerSec   float64 `json:"peak_messages_per_sec"`
	LowestMessagesPerSec float64 `json:"lowest_messages_per_sec"`
	ErrorCount           int64   `json:"error_count"`
	SuccessRate          float64 `json:"success_rate"`
	Intervals            []IntervalMetrics `json:"intervals"`
}

type IntervalMetrics struct {
	IntervalStart    int64   `json:"interval_start"`
	IntervalEnd      int64   `json:"interval_end"`
	MessagesSent     int64   `json:"messages_sent"`
	MessagesReceived int64   `json:"messages_received"`
	MessagesPerSec   float64 `json:"messages_per_sec"`
	ErrorsInInterval int64   `json:"errors_in_interval"`
}

var (
	messagesSent     int64
	messagesReceived int64
	errorCount       int64
	intervals        []IntervalMetrics
	intervalMutex    sync.Mutex
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("🔥 STRESS TEST PROLONGADO: 10+ minutos - FlexCore")

	// Configurar cliente Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	})

	// Teste de conexão
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis connection failed: %v", err)
	}

	log.Printf("✅ Redis connected - Starting 10-minute stress test...")

	// Configurar métricas
	metrics := &StressTestMetrics{
		TestStartedAt:        time.Now().Unix(),
		LowestMessagesPerSec: 1000000, // High initial value
	}

	// Canal para parar o teste
	stopChan := make(chan bool)

	// Goroutine para enviar mensagens
	go sendMessagesWorker(rdb, stopChan)

	// Goroutine para receber mensagens
	go receiveMessagesWorker(rdb, stopChan)

	// Goroutine para coletar métricas a cada 30 segundos
	go metricsCollector(stopChan)

	// Executar por 10 minutos (600 segundos)
	testDuration := 10 * time.Minute
	log.Printf("⏰ Test will run for %v", testDuration)

	time.Sleep(testDuration)

	// Parar todos os workers
	close(stopChan)
	time.Sleep(2 * time.Second) // Wait for workers to finish

	// Calcular métricas finais
	metrics.TestCompletedAt = time.Now().Unix()
	metrics.TestDurationSeconds = metrics.TestCompletedAt - metrics.TestStartedAt
	metrics.TotalMessagesSent = atomic.LoadInt64(&messagesSent)
	metrics.TotalMessagesReceived = atomic.LoadInt64(&messagesReceived)
	metrics.ErrorCount = atomic.LoadInt64(&errorCount)

	if metrics.TestDurationSeconds > 0 {
		metrics.AverageMessagesPerSec = float64(metrics.TotalMessagesSent) / float64(metrics.TestDurationSeconds)
	}

	if metrics.TotalMessagesSent > 0 {
		metrics.SuccessRate = (float64(metrics.TotalMessagesReceived) / float64(metrics.TotalMessagesSent)) * 100
	}

	// Calcular pico e menor taxa
	intervalMutex.Lock()
	for _, interval := range intervals {
		if interval.MessagesPerSec > metrics.PeakMessagesPerSec {
			metrics.PeakMessagesPerSec = interval.MessagesPerSec
		}
		if interval.MessagesPerSec < metrics.LowestMessagesPerSec && interval.MessagesPerSec > 0 {
			metrics.LowestMessagesPerSec = interval.MessagesPerSec
		}
	}
	metrics.Intervals = intervals
	intervalMutex.Unlock()

	// Salvar resultados
	resultsJSON, _ := json.MarshalIndent(metrics, "", "  ")
	filename := fmt.Sprintf("/home/marlonsc/flext/flexcore/stress_test_10min_%d.json", metrics.TestStartedAt)
	os.WriteFile(filename, resultsJSON, 0644)

	// Relatório final
	log.Printf("")
	log.Printf("🎯 STRESS TEST 10 MINUTOS COMPLETADO:")
	log.Printf("   ⏱️  Duração: %d segundos (%.1f minutos)", metrics.TestDurationSeconds, float64(metrics.TestDurationSeconds)/60)
	log.Printf("   📤 Mensagens enviadas: %d", metrics.TotalMessagesSent)
	log.Printf("   📥 Mensagens recebidas: %d", metrics.TotalMessagesReceived)
	log.Printf("   📊 Taxa média: %.2f msgs/sec", metrics.AverageMessagesPerSec)
	log.Printf("   🔥 Pico de taxa: %.2f msgs/sec", metrics.PeakMessagesPerSec)
	log.Printf("   🐌 Menor taxa: %.2f msgs/sec", metrics.LowestMessagesPerSec)
	log.Printf("   ✅ Taxa de sucesso: %.2f%%", metrics.SuccessRate)
	log.Printf("   ❌ Erros: %d", metrics.ErrorCount)
	log.Printf("   📊 Intervalos coletados: %d", len(metrics.Intervals))
	log.Printf("   💾 Resultados salvos: %s", filename)

	// Verificar se passou no teste de qualidade
	if metrics.TestDurationSeconds >= 600 && metrics.AverageMessagesPerSec >= 10 && metrics.SuccessRate >= 95 {
		log.Printf("✅ STRESS TEST PASSOU - Sistema estável por 10+ minutos!")
	} else {
		log.Printf("⚠️  STRESS TEST ATENÇÃO - Verificar métricas:")
		if metrics.TestDurationSeconds < 600 {
			log.Printf("    ⏱️  Duração menor que 10 minutos")
		}
		if metrics.AverageMessagesPerSec < 10 {
			log.Printf("    📊 Taxa média baixa (< 10 msgs/sec)")
		}
		if metrics.SuccessRate < 95 {
			log.Printf("    ❌ Taxa de sucesso baixa (< 95%%)")
		}
	}

	rdb.Close()
}

func sendMessagesWorker(rdb *redis.Client, stopChan chan bool) {
	ctx := context.Background()
	ticker := time.NewTicker(100 * time.Millisecond) // Send every 100ms
	defer ticker.Stop()

	messageID := 1
	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// Send batch of messages
			for i := 0; i < 5; i++ { // Send 5 messages per batch = 50 msgs/sec
				message := fmt.Sprintf(`{
					"id": %d,
					"timestamp": %d,
					"data": "stress_test_message_%d",
					"worker": "sender",
					"batch": %d
				}`, messageID, time.Now().Unix(), messageID, i)

				if err := rdb.LPush(ctx, "flexcore:stress:messages", message).Err(); err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&messagesSent, 1)
				}
				messageID++
			}
		}
	}
}

func receiveMessagesWorker(rdb *redis.Client, stopChan chan bool) {
	ctx := context.Background()
	ticker := time.NewTicker(50 * time.Millisecond) // Check every 50ms
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// Receive batch of messages
			for i := 0; i < 10; i++ { // Try to receive up to 10 messages
				result, err := rdb.RPop(ctx, "flexcore:stress:messages").Result()
				if err == redis.Nil {
					break // No more messages
				} else if err != nil {
					atomic.AddInt64(&errorCount, 1)
					break
				} else if result != "" {
					atomic.AddInt64(&messagesReceived, 1)
				}
			}
		}
	}
}

func metricsCollector(stopChan chan bool) {
	ticker := time.NewTicker(30 * time.Second) // Collect metrics every 30 seconds
	defer ticker.Stop()

	lastSent := int64(0)
	lastReceived := int64(0)
	lastErrors := int64(0)

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			currentTime := time.Now().Unix()
			currentSent := atomic.LoadInt64(&messagesSent)
			currentReceived := atomic.LoadInt64(&messagesReceived)
			currentErrors := atomic.LoadInt64(&errorCount)

			intervalSent := currentSent - lastSent
			intervalReceived := currentReceived - lastReceived
			intervalErrors := currentErrors - lastErrors

			messagesPerSec := float64(intervalSent) / 30.0 // 30 second interval

			interval := IntervalMetrics{
				IntervalStart:    currentTime - 30,
				IntervalEnd:      currentTime,
				MessagesSent:     intervalSent,
				MessagesReceived: intervalReceived,
				MessagesPerSec:   messagesPerSec,
				ErrorsInInterval: intervalErrors,
			}

			intervalMutex.Lock()
			intervals = append(intervals, interval)
			intervalMutex.Unlock()

			log.Printf("📊 Interval [%d]: %d sent, %d received, %.2f msgs/sec, %d errors",
				len(intervals), intervalSent, intervalReceived, messagesPerSec, intervalErrors)

			lastSent = currentSent
			lastReceived = currentReceived
			lastErrors = currentErrors
		}
	}
}
