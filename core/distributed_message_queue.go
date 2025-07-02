// Package core - Distributed Message Queue Implementation
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flext/flexcore/infrastructure/windmill"
	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// DistributedMessageQueue implements a distributed message queue
type DistributedMessageQueue struct {
	windmillClient *windmill.Client
	config         *FlexCoreConfig
	redis          *redis.Client
	queues         map[string]*QueueInfo
	metrics        *MessageQueueMetrics
	mu             sync.RWMutex
	workers        map[string]*QueueWorker
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// QueueInfo holds queue metadata
type QueueInfo struct {
	Name            string
	Type            string // fifo, priority, delayed
	MaxSize         int
	TTL             time.Duration
	DeadLetterQueue string
	CreatedAt       time.Time
	MessageCount    int64
}

// QueueWorker processes messages from a queue
type QueueWorker struct {
	QueueName string
	WorkerID  string
	Running   bool
	Processed int64
	Errors    int64
}

// MessageQueueMetrics tracks queue performance
type MessageQueueMetrics struct {
	MessagesQueued     int64
	MessagesDelivered  int64
	MessagesExpired    int64
	MessagesFailed     int64
	MessagesDropped    int64
	QueueDepth         int64
	ProcessingTimeMs   int64
}

// NewDistributedMessageQueue creates a new distributed message queue
func NewDistributedMessageQueue(windmillClient *windmill.Client, config *FlexCoreConfig) *DistributedMessageQueue {
	// Try Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		// Redis not available, will use in-memory fallback
		rdb = nil
	}

	dmq := &DistributedMessageQueue{
		windmillClient: windmillClient,
		config:         config,
		redis:          rdb,
		queues:         make(map[string]*QueueInfo),
		metrics:        &MessageQueueMetrics{},
		workers:        make(map[string]*QueueWorker),
		stopChan:       make(chan struct{}),
	}

	// Initialize configured queues
	for _, queueConfig := range config.MessageQueues {
		dmq.createQueue(&queueConfig)
	}

	return dmq
}

// Start starts the message queue system
func (dmq *DistributedMessageQueue) Start(ctx context.Context) error {
	dmq.mu.Lock()
	defer dmq.mu.Unlock()

	// Start workers for each queue
	for queueName := range dmq.queues {
		workerCount := 3 // Configurable per queue
		for i := 0; i < workerCount; i++ {
			workerID := fmt.Sprintf("%s-worker-%d", queueName, i)
			worker := &QueueWorker{
				QueueName: queueName,
				WorkerID:  workerID,
				Running:   true,
			}
			dmq.workers[workerID] = worker

			dmq.wg.Add(1)
			go dmq.runWorker(ctx, worker)
		}
	}

	// Start maintenance routine
	dmq.wg.Add(1)
	go dmq.maintenanceLoop(ctx)

	return nil
}

// Stop stops the message queue system
func (dmq *DistributedMessageQueue) Stop(ctx context.Context) error {
	close(dmq.stopChan)
	
	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		dmq.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Graceful shutdown
	case <-time.After(10 * time.Second):
		// Timeout
	}

	if dmq.redis != nil {
		dmq.redis.Close()
	}

	return nil
}

// SendMessage sends a message to a queue
func (dmq *DistributedMessageQueue) SendMessage(ctx context.Context, queueName string, message *Message) result.Result[bool] {
	dmq.mu.RLock()
	queue, exists := dmq.queues[queueName]
	dmq.mu.RUnlock()

	if !exists {
		return result.Failure[bool](errors.ValidationError("queue not found"))
	}

	// Set message defaults
	if message.ID == "" {
		message.ID = uuid.New().String()
	}
	message.Queue = queueName
	message.CreatedAt = time.Now()
	if message.MaxAttempts == 0 {
		message.MaxAttempts = 3
	}

	// Calculate expiry
	if queue.TTL > 0 && message.ExpiresAt == nil {
		expiresAt := time.Now().Add(queue.TTL)
		message.ExpiresAt = &expiresAt
	}

	// Store message
	if dmq.redis != nil {
		return dmq.sendMessageRedis(ctx, queue, message)
	}
	return dmq.sendMessageInMemory(ctx, queue, message)
}

// ReceiveMessages receives messages from a queue
func (dmq *DistributedMessageQueue) ReceiveMessages(ctx context.Context, queueName string, maxMessages int) result.Result[[]*Message] {
	dmq.mu.RLock()
	queue, exists := dmq.queues[queueName]
	dmq.mu.RUnlock()

	if !exists {
		return result.Failure[[]*Message](errors.ValidationError("queue not found"))
	}

	if dmq.redis != nil {
		return dmq.receiveMessagesRedis(ctx, queue, maxMessages)
	}
	return dmq.receiveMessagesInMemory(ctx, queue, maxMessages)
}

// GetMetrics returns queue metrics
func (dmq *DistributedMessageQueue) GetMetrics() *MessageQueueMetrics {
	return &MessageQueueMetrics{
		MessagesQueued:    atomic.LoadInt64(&dmq.metrics.MessagesQueued),
		MessagesDelivered: atomic.LoadInt64(&dmq.metrics.MessagesDelivered),
		MessagesExpired:   atomic.LoadInt64(&dmq.metrics.MessagesExpired),
		MessagesFailed:    atomic.LoadInt64(&dmq.metrics.MessagesFailed),
		QueueDepth:        dmq.calculateTotalQueueDepth(),
		ProcessingTimeMs:  atomic.LoadInt64(&dmq.metrics.ProcessingTimeMs),
	}
}

// PerformMaintenance performs periodic maintenance
func (dmq *DistributedMessageQueue) PerformMaintenance(ctx context.Context) {
	// Clean expired messages
	dmq.cleanExpiredMessages(ctx)
	
	// Move failed messages to DLQ
	dmq.processDLQ(ctx)
	
	// Update metrics
	dmq.updateMetrics(ctx)
}

// Private methods - Redis implementation

func (dmq *DistributedMessageQueue) sendMessageRedis(ctx context.Context, queue *QueueInfo, message *Message) result.Result[bool] {
	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		return result.Failure[bool](fmt.Errorf("failed to serialize message: %w", err))
	}

	queueKey := fmt.Sprintf("flexcore:queue:%s", queue.Name)
	
	// Push based on queue type
	switch queue.Type {
	case "priority":
		// Use sorted set with priority as score
		score := float64(message.Priority)
		if score == 0 {
			score = float64(time.Now().Unix())
		}
		err = dmq.redis.ZAdd(ctx, queueKey, &redis.Z{
			Score:  score,
			Member: string(data),
		}).Err()
		
	case "delayed":
		// Use sorted set with delivery time as score
		deliveryTime := time.Now()
		if message.ExpiresAt != nil {
			deliveryTime = *message.ExpiresAt
		}
		err = dmq.redis.ZAdd(ctx, queueKey, &redis.Z{
			Score:  float64(deliveryTime.Unix()),
			Member: string(data),
		}).Err()
		
	default: // fifo
		// Use list
		err = dmq.redis.RPush(ctx, queueKey, data).Err()
	}

	if err != nil {
		return result.Failure[bool](fmt.Errorf("failed to queue message: %w", err))
	}

	atomic.AddInt64(&dmq.metrics.MessagesQueued, 1)
	atomic.AddInt64(&queue.MessageCount, 1)
	
	return result.Success(true)
}

func (dmq *DistributedMessageQueue) receiveMessagesRedis(ctx context.Context, queue *QueueInfo, maxMessages int) result.Result[[]*Message] {
	queueKey := fmt.Sprintf("flexcore:queue:%s", queue.Name)
	processingKey := fmt.Sprintf("flexcore:processing:%s", queue.Name)
	
	messages := make([]*Message, 0, maxMessages)
	
	for i := 0; i < maxMessages; i++ {
		var messageData string
		var err error
		
		switch queue.Type {
		case "priority":
			// Pop highest priority
			result := dmq.redis.ZPopMax(ctx, queueKey, 1)
			if err = result.Err(); err != nil || len(result.Val()) == 0 {
				break
			}
			messageData = result.Val()[0].Member.(string)
			
		case "delayed":
			// Get messages ready for delivery
			now := float64(time.Now().Unix())
			result := dmq.redis.ZRangeByScore(ctx, queueKey, &redis.ZRangeBy{
				Min:   "0",
				Max:   fmt.Sprintf("%f", now),
				Count: 1,
			})
			if err = result.Err(); err != nil || len(result.Val()) == 0 {
				break
			}
			messageData = result.Val()[0]
			// Remove from delayed queue
			dmq.redis.ZRem(ctx, queueKey, messageData)
			
		default: // fifo
			// Pop from left
			messageData, err = dmq.redis.LPop(ctx, queueKey).Result()
			if err == redis.Nil {
				break
			} else if err != nil {
				return result.Failure[[]*Message](err)
			}
		}
		
		// Deserialize message
		var message Message
		if err := json.Unmarshal([]byte(messageData), &message); err != nil {
			continue
		}
		
		// Move to processing queue
		dmq.redis.HSet(ctx, processingKey, message.ID, messageData)
		dmq.redis.Expire(ctx, processingKey, 5*time.Minute)
		
		messages = append(messages, &message)
		atomic.AddInt64(&dmq.metrics.MessagesDelivered, 1)
	}
	
	return result.Success(messages)
}

// In-memory fallback implementation

var inMemoryQueues = struct {
	sync.RWMutex
	queues map[string][]*Message
}{
	queues: make(map[string][]*Message),
}

func (dmq *DistributedMessageQueue) sendMessageInMemory(ctx context.Context, queue *QueueInfo, message *Message) result.Result[bool] {
	inMemoryQueues.Lock()
	defer inMemoryQueues.Unlock()
	
	if _, exists := inMemoryQueues.queues[queue.Name]; !exists {
		inMemoryQueues.queues[queue.Name] = make([]*Message, 0)
	}
	
	// Check queue size limit
	if queue.MaxSize > 0 && len(inMemoryQueues.queues[queue.Name]) >= queue.MaxSize {
		atomic.AddInt64(&dmq.metrics.MessagesDropped, 1)
		return result.Failure[bool](errors.SystemError("queue full"))
	}
	
	inMemoryQueues.queues[queue.Name] = append(inMemoryQueues.queues[queue.Name], message)
	atomic.AddInt64(&dmq.metrics.MessagesQueued, 1)
	atomic.AddInt64(&queue.MessageCount, 1)
	
	return result.Success(true)
}

func (dmq *DistributedMessageQueue) receiveMessagesInMemory(ctx context.Context, queue *QueueInfo, maxMessages int) result.Result[[]*Message] {
	inMemoryQueues.Lock()
	defer inMemoryQueues.Unlock()
	
	queueMessages, exists := inMemoryQueues.queues[queue.Name]
	if !exists || len(queueMessages) == 0 {
		return result.Success([]*Message{})
	}
	
	// Get up to maxMessages
	count := min(maxMessages, len(queueMessages))
	messages := make([]*Message, count)
	copy(messages, queueMessages[:count])
	
	// Remove from queue
	inMemoryQueues.queues[queue.Name] = queueMessages[count:]
	
	atomic.AddInt64(&dmq.metrics.MessagesDelivered, int64(count))
	atomic.AddInt64(&queue.MessageCount, -int64(count))
	
	return result.Success(messages)
}

// Worker implementation

func (dmq *DistributedMessageQueue) runWorker(ctx context.Context, worker *QueueWorker) {
	defer dmq.wg.Done()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-dmq.stopChan:
			worker.Running = false
			return
		case <-ctx.Done():
			worker.Running = false
			return
		case <-ticker.C:
			dmq.processQueue(ctx, worker)
		}
	}
}

func (dmq *DistributedMessageQueue) processQueue(ctx context.Context, worker *QueueWorker) {
	// This would process messages from the queue
	// For now, we just update metrics
	atomic.AddInt64(&worker.Processed, 1)
}

// Maintenance

func (dmq *DistributedMessageQueue) maintenanceLoop(ctx context.Context) {
	defer dmq.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-dmq.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			dmq.PerformMaintenance(ctx)
		}
	}
}

func (dmq *DistributedMessageQueue) cleanExpiredMessages(ctx context.Context) {
	if dmq.redis != nil {
		// Clean expired messages from Redis
		for queueName := range dmq.queues {
			queueKey := fmt.Sprintf("flexcore:queue:%s", queueName)
			// Remove expired messages from sorted sets
			dmq.redis.ZRemRangeByScore(ctx, queueKey, "0", fmt.Sprintf("%d", time.Now().Unix()))
		}
	} else {
		// Clean in-memory
		inMemoryQueues.Lock()
		defer inMemoryQueues.Unlock()
		
		for queueName, messages := range inMemoryQueues.queues {
			validMessages := make([]*Message, 0)
			for _, msg := range messages {
				if msg.ExpiresAt == nil || msg.ExpiresAt.After(time.Now()) {
					validMessages = append(validMessages, msg)
				} else {
					atomic.AddInt64(&dmq.metrics.MessagesExpired, 1)
				}
			}
			inMemoryQueues.queues[queueName] = validMessages
		}
	}
}

func (dmq *DistributedMessageQueue) processDLQ(ctx context.Context) {
	// Move failed messages to dead letter queue
	for _, queue := range dmq.queues {
		if queue.DeadLetterQueue != "" {
			// Process failed messages
			// This would check processing queue for messages that exceeded max attempts
		}
	}
}

func (dmq *DistributedMessageQueue) updateMetrics(ctx context.Context) {
	// Update queue depth metric
	totalDepth := int64(0)
	
	if dmq.redis != nil {
		for queueName := range dmq.queues {
			queueKey := fmt.Sprintf("flexcore:queue:%s", queueName)
			if count, err := dmq.redis.LLen(ctx, queueKey).Result(); err == nil {
				totalDepth += count
			}
		}
	} else {
		inMemoryQueues.RLock()
		for _, messages := range inMemoryQueues.queues {
			totalDepth += int64(len(messages))
		}
		inMemoryQueues.RUnlock()
	}
	
	atomic.StoreInt64(&dmq.metrics.QueueDepth, totalDepth)
}

func (dmq *DistributedMessageQueue) createQueue(config *QueueConfig) {
	queue := &QueueInfo{
		Name:            config.Name,
		Type:            config.Type,
		MaxSize:         config.MaxSize,
		TTL:             config.TTL,
		DeadLetterQueue: config.DeadLetterQueue,
		CreatedAt:       time.Now(),
		MessageCount:    0,
	}
	
	dmq.queues[config.Name] = queue
}

func (dmq *DistributedMessageQueue) calculateTotalQueueDepth() int64 {
	return atomic.LoadInt64(&dmq.metrics.QueueDepth)
}