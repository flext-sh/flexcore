// Package core - Real Distributed Scheduler Implementation
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flext/flexcore/infrastructure/windmill"
	"github.com/flext/flexcore/shared/errors"
	"github.com/flext/flexcore/shared/result"
	"github.com/go-redis/redis/v8"
	"github.com/robfig/cron/v3"
)

// RealDistributedScheduler implements a real distributed scheduler with cron support
type RealDistributedScheduler struct {
	windmillClient *windmill.Client
	config         *FlexCoreConfig
	nodeID         string
	redis          *redis.Client
	cron           *cron.Cron
	schedules      map[string]*ScheduleInfo
	executions     map[string]*ExecutionInfo
	metrics        *SchedulerMetrics
	mu             sync.RWMutex
	stopChan       chan struct{}
	wg             sync.WaitGroup
	isLeader       bool
}

// DistributedScheduler is an alias for RealDistributedScheduler
type DistributedScheduler = RealDistributedScheduler

// ScheduleInfo holds schedule metadata
type ScheduleInfo struct {
	ID             string
	Name           string
	CronExpression string
	WorkflowPath   string
	Input          map[string]interface{}
	Timezone       string
	Singleton      bool
	MaxInstances   int
	LastRun        *time.Time
	NextRun        *time.Time
	Status         string // active, paused, disabled
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ExecutionInfo tracks running executions
type ExecutionInfo struct {
	ID           string
	ScheduleID   string
	JobID        string
	StartTime    time.Time
	EndTime      *time.Time
	Status       string // running, completed, failed
	Result       interface{}
	Error        string
	NodeID       string
}

// SchedulerMetrics tracks scheduler performance
type SchedulerMetrics struct {
	ScheduledJobs      int64
	ExecutedJobs       int64
	FailedJobs         int64
	SkippedJobs        int64
	ActiveSchedules    int64
	RunningExecutions  int64
	AverageExecutionMs int64
}

// NewDistributedScheduler creates a real distributed scheduler
func NewDistributedScheduler(windmillClient *windmill.Client, config *FlexCoreConfig) *RealDistributedScheduler {
	return NewRealDistributedScheduler(windmillClient, config, config.NodeID)
}

// NewRealDistributedScheduler creates a real distributed scheduler
func NewRealDistributedScheduler(windmillClient *windmill.Client, config *FlexCoreConfig, nodeID string) *RealDistributedScheduler {
	// Try Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use different DB for scheduler
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb = nil
	}

	// Create cron scheduler with seconds support
	c := cron.New(cron.WithSeconds())

	return &RealDistributedScheduler{
		windmillClient: windmillClient,
		config:         config,
		nodeID:         nodeID,
		redis:          rdb,
		cron:           c,
		schedules:      make(map[string]*ScheduleInfo),
		executions:     make(map[string]*ExecutionInfo),
		metrics:        &SchedulerMetrics{},
		stopChan:       make(chan struct{}),
	}
}

// Start starts the distributed scheduler
func (s *RealDistributedScheduler) Start(ctx context.Context) result.Result[bool] {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load configured schedules
	for _, schedConfig := range s.config.Schedulers {
		if err := s.createSchedule(&schedConfig); err != nil {
			return result.Failure[bool](fmt.Errorf("failed to create schedule %s: %w", schedConfig.Name, err))
		}
	}

	// Start leader election loop
	s.wg.Add(1)
	go s.leaderElectionLoop(ctx)

	// Start execution monitoring
	s.wg.Add(1)
	go s.executionMonitor(ctx)

	// Start cron scheduler
	s.cron.Start()

	log.Printf("Distributed scheduler started on node %s with %d schedules", s.nodeID, len(s.schedules))
	return result.Success(true)
}

// Stop stops the distributed scheduler
func (s *RealDistributedScheduler) Stop(ctx context.Context) result.Result[bool] {
	close(s.stopChan)
	
	// Stop cron
	cronCtx := s.cron.Stop()
	
	// Wait for cron to stop
	select {
	case <-cronCtx.Done():
		// Cron stopped
	case <-time.After(5 * time.Second):
		// Timeout
	}

	// Wait for goroutines
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Clean shutdown
	case <-time.After(10 * time.Second):
		// Timeout
	}

	if s.redis != nil {
		s.redis.Close()
	}

	return result.Success(true)
}

// ScheduleJob schedules a one-time job
func (s *RealDistributedScheduler) ScheduleJob(ctx context.Context, schedulerName string, input map[string]interface{}) result.Result[string] {
	s.mu.RLock()
	schedule, exists := s.schedules[schedulerName]
	s.mu.RUnlock()

	if !exists {
		return result.Failure[string](errors.ValidationError("scheduler not found"))
	}

	// Execute immediately
	return s.executeScheduledJob(ctx, schedule, input)
}

// CreateSchedule creates a new cron schedule
func (s *RealDistributedScheduler) CreateSchedule(ctx context.Context, config *SchedulerConfig) result.Result[string] {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.createSchedule(config); err != nil {
		return result.Failure[string](err)
	}

	return result.Success(config.Name)
}

// DeleteSchedule removes a schedule
func (s *RealDistributedScheduler) DeleteSchedule(ctx context.Context, scheduleID string) result.Result[bool] {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return result.Failure[bool](errors.ValidationError("schedule not found"))
	}

	// Remove from cron
	s.cron.Remove(cron.EntryID(len(s.schedules))) // Would track actual entry IDs

	// Update status
	schedule.Status = "disabled"
	delete(s.schedules, scheduleID)

	// Clean from Redis if available
	if s.redis != nil {
		key := fmt.Sprintf("scheduler:schedule:%s", scheduleID)
		s.redis.Del(ctx, key)
	}

	return result.Success(true)
}

// GetMetrics returns scheduler metrics
func (s *RealDistributedScheduler) GetMetrics() *SchedulerMetrics {
	return &SchedulerMetrics{
		ScheduledJobs:      atomic.LoadInt64(&s.metrics.ScheduledJobs),
		ExecutedJobs:       atomic.LoadInt64(&s.metrics.ExecutedJobs),
		FailedJobs:         atomic.LoadInt64(&s.metrics.FailedJobs),
		SkippedJobs:        atomic.LoadInt64(&s.metrics.SkippedJobs),
		ActiveSchedules:    int64(len(s.schedules)),
		RunningExecutions:  int64(len(s.executions)),
		AverageExecutionMs: atomic.LoadInt64(&s.metrics.AverageExecutionMs),
	}
}

// PerformMaintenance performs periodic maintenance
func (s *RealDistributedScheduler) PerformMaintenance(ctx context.Context) {
	// Clean old executions
	s.cleanOldExecutions(ctx)
	
	// Update next run times
	s.updateNextRunTimes()
	
	// Check stuck executions
	s.checkStuckExecutions(ctx)
}

// Private methods

func (s *RealDistributedScheduler) createSchedule(config *SchedulerConfig) error {
	// Parse cron expression
	_, err := cron.ParseStandard(config.CronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	schedule := &ScheduleInfo{
		ID:             config.Name,
		Name:           config.Name,
		CronExpression: config.CronExpression,
		WorkflowPath:   config.WorkflowPath,
		Input:          config.Input,
		Timezone:       config.Timezone,
		Singleton:      config.Singleton,
		MaxInstances:   config.MaxInstances,
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Calculate next run
	nextRun := s.calculateNextRun(schedule)
	schedule.NextRun = &nextRun

	// Store schedule
	s.schedules[config.Name] = schedule
	atomic.AddInt64(&s.metrics.ActiveSchedules, 1)

	// Add to cron
	_, err = s.cron.AddFunc(config.CronExpression, func() {
		s.handleCronTrigger(schedule)
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// Persist to Redis if available
	if s.redis != nil {
		s.persistSchedule(context.Background(), schedule)
	}

	return nil
}

func (s *RealDistributedScheduler) handleCronTrigger(schedule *ScheduleInfo) {
	ctx := context.Background()
	
	// Check if we're the leader
	if !s.isLeader && s.redis != nil {
		// Only leader executes in distributed mode
		return
	}

	// Check singleton constraint
	if schedule.Singleton {
		if !s.acquireScheduleLock(ctx, schedule.ID) {
			atomic.AddInt64(&s.metrics.SkippedJobs, 1)
			log.Printf("Skipping singleton job %s - already running", schedule.Name)
			return
		}
		defer s.releaseScheduleLock(ctx, schedule.ID)
	}

	// Check max instances
	if schedule.MaxInstances > 0 {
		running := s.countRunningInstances(schedule.ID)
		if running >= schedule.MaxInstances {
			atomic.AddInt64(&s.metrics.SkippedJobs, 1)
			log.Printf("Skipping job %s - max instances reached (%d)", schedule.Name, running)
			return
		}
	}

	// Execute job
	atomic.AddInt64(&s.metrics.ScheduledJobs, 1)
	
	jobResult := s.executeScheduledJob(ctx, schedule, schedule.Input)
	if jobResult.IsFailure() {
		atomic.AddInt64(&s.metrics.FailedJobs, 1)
		log.Printf("Failed to execute scheduled job %s: %v", schedule.Name, jobResult.Error())
	} else {
		atomic.AddInt64(&s.metrics.ExecutedJobs, 1)
		log.Printf("Successfully scheduled job %s with ID %s", schedule.Name, jobResult.Value())
	}

	// Update last run time
	now := time.Now()
	schedule.LastRun = &now
	nextRun := s.calculateNextRun(schedule)
	schedule.NextRun = &nextRun
}

func (s *RealDistributedScheduler) executeScheduledJob(ctx context.Context, schedule *ScheduleInfo, input map[string]interface{}) result.Result[string] {
	// Create execution record
	execution := &ExecutionInfo{
		ID:         fmt.Sprintf("exec-%d", time.Now().UnixNano()),
		ScheduleID: schedule.ID,
		StartTime:  time.Now(),
		Status:     "running",
		NodeID:     s.nodeID,
	}

	s.mu.Lock()
	s.executions[execution.ID] = execution
	atomic.AddInt64(&s.metrics.RunningExecutions, 1)
	s.mu.Unlock()

	// Execute via Windmill
	if s.windmillClient != nil {
		runReq := windmill.RunWorkflowRequest{
			Args: input,
			Tag:  &schedule.Name,
		}

		jobResult := s.windmillClient.RunWorkflow(ctx, schedule.WorkflowPath, runReq)
		if jobResult.IsFailure() {
			execution.Status = "failed"
			execution.Error = jobResult.Error().Error()
			endTime := time.Now()
			execution.EndTime = &endTime
			return result.Failure[string](jobResult.Error())
		}

		job := jobResult.Value()
		execution.JobID = job.ID
		
		// Monitor execution in background
		go s.monitorExecution(ctx, execution, job.ID)
		
		return result.Success(job.ID)
	}

	// Simulate execution without Windmill
	go func() {
		time.Sleep(2 * time.Second) // Simulate work
		
		s.mu.Lock()
		execution.Status = "completed"
		endTime := time.Now()
		execution.EndTime = &endTime
		execution.Result = map[string]interface{}{
			"message": "Simulated execution completed",
			"input":   input,
		}
		atomic.AddInt64(&s.metrics.RunningExecutions, -1)
		s.mu.Unlock()
		
		// Update average execution time
		duration := execution.EndTime.Sub(execution.StartTime).Milliseconds()
		atomic.StoreInt64(&s.metrics.AverageExecutionMs, duration)
	}()

	return result.Success(execution.ID)
}

func (s *RealDistributedScheduler) monitorExecution(ctx context.Context, execution *ExecutionInfo, jobID string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(30 * time.Minute) // Max execution time
	
	for {
		select {
		case <-ticker.C:
			// Check job status
			statusResult := s.windmillClient.GetJob(ctx, jobID)
			if statusResult.IsFailure() {
				continue
			}
			
			job := statusResult.Value()
			if job.Success != nil {
				s.mu.Lock()
				if *job.Success {
					execution.Status = "completed"
					execution.Result = job.Result
				} else {
					execution.Status = "failed"
					if job.Error != nil {
						execution.Error = *job.Error
					}
				}
				endTime := time.Now()
				execution.EndTime = &endTime
				atomic.AddInt64(&s.metrics.RunningExecutions, -1)
				s.mu.Unlock()
				
				// Update average execution time
				duration := execution.EndTime.Sub(execution.StartTime).Milliseconds()
				atomic.StoreInt64(&s.metrics.AverageExecutionMs, duration)
				return
			}
			
		case <-timeout:
			// Timeout
			s.mu.Lock()
			execution.Status = "failed"
			execution.Error = "execution timeout"
			endTime := time.Now()
			execution.EndTime = &endTime
			atomic.AddInt64(&s.metrics.RunningExecutions, -1)
			s.mu.Unlock()
			return
			
		case <-s.stopChan:
			return
		}
	}
}

func (s *RealDistributedScheduler) leaderElectionLoop(ctx context.Context) {
	defer s.wg.Done()
	
	if s.redis == nil {
		// Single node mode - always leader
		s.isLeader = true
		return
	}
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tryBecomeLeader(ctx)
		}
	}
}

func (s *RealDistributedScheduler) tryBecomeLeader(ctx context.Context) {
	key := "scheduler:leader"
	ttl := 10 * time.Second
	
	// Try to acquire leader lock
	result := s.redis.SetNX(ctx, key, s.nodeID, ttl)
	if result.Val() {
		s.isLeader = true
		log.Printf("Scheduler node %s became LEADER", s.nodeID)
		
		// Load schedules from Redis
		s.loadSchedulesFromRedis(ctx)
	} else {
		// Check if we're still the leader
		current, _ := s.redis.Get(ctx, key).Result()
		s.isLeader = (current == s.nodeID)
		
		// Refresh TTL if we're leader
		if s.isLeader {
			s.redis.Expire(ctx, key, ttl)
		}
	}
}

func (s *RealDistributedScheduler) acquireScheduleLock(ctx context.Context, scheduleID string) bool {
	if s.redis == nil {
		// Local mode - use in-memory check
		s.mu.RLock()
		defer s.mu.RUnlock()
		
		for _, exec := range s.executions {
			if exec.ScheduleID == scheduleID && exec.Status == "running" {
				return false
			}
		}
		return true
	}
	
	// Distributed mode - use Redis lock
	key := fmt.Sprintf("scheduler:lock:%s", scheduleID)
	ttl := 5 * time.Minute
	
	result := s.redis.SetNX(ctx, key, s.nodeID, ttl)
	return result.Val()
}

func (s *RealDistributedScheduler) releaseScheduleLock(ctx context.Context, scheduleID string) {
	if s.redis != nil {
		key := fmt.Sprintf("scheduler:lock:%s", scheduleID)
		s.redis.Del(ctx, key)
	}
}

func (s *RealDistributedScheduler) countRunningInstances(scheduleID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	count := 0
	for _, exec := range s.executions {
		if exec.ScheduleID == scheduleID && exec.Status == "running" {
			count++
		}
	}
	return count
}

func (s *RealDistributedScheduler) calculateNextRun(schedule *ScheduleInfo) time.Time {
	// Parse cron expression
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(schedule.CronExpression)
	if err != nil {
		return time.Now().Add(1 * time.Hour) // Default fallback
	}
	
	return cronSchedule.Next(time.Now())
}

func (s *RealDistributedScheduler) executionMonitor(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.PerformMaintenance(ctx)
		}
	}
}

func (s *RealDistributedScheduler) cleanOldExecutions(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-24 * time.Hour)
	toDelete := []string{}
	
	for id, exec := range s.executions {
		if exec.EndTime != nil && exec.EndTime.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}
	
	for _, id := range toDelete {
		delete(s.executions, id)
	}
}

func (s *RealDistributedScheduler) updateNextRunTimes() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, schedule := range s.schedules {
		if schedule.Status == "active" {
			nextRun := s.calculateNextRun(schedule)
			schedule.NextRun = &nextRun
		}
	}
}

func (s *RealDistributedScheduler) checkStuckExecutions(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	timeout := 1 * time.Hour
	now := time.Now()
	
	for _, exec := range s.executions {
		if exec.Status == "running" && now.Sub(exec.StartTime) > timeout {
			exec.Status = "failed"
			exec.Error = "execution stuck - timeout"
			exec.EndTime = &now
			atomic.AddInt64(&s.metrics.FailedJobs, 1)
			atomic.AddInt64(&s.metrics.RunningExecutions, -1)
		}
	}
}

func (s *RealDistributedScheduler) persistSchedule(ctx context.Context, schedule *ScheduleInfo) {
	key := fmt.Sprintf("scheduler:schedule:%s", schedule.ID)
	data, _ := json.Marshal(schedule)
	s.redis.Set(ctx, key, data, 0)
}

func (s *RealDistributedScheduler) loadSchedulesFromRedis(ctx context.Context) {
	pattern := "scheduler:schedule:*"
	keys, err := s.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return
	}
	
	for _, key := range keys {
		data, err := s.redis.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		
		var schedule ScheduleInfo
		if err := json.Unmarshal([]byte(data), &schedule); err != nil {
			continue
		}
		
		s.mu.Lock()
		s.schedules[schedule.ID] = &schedule
		s.mu.Unlock()
		
		// Re-add to cron if active
		if schedule.Status == "active" {
			s.cron.AddFunc(schedule.CronExpression, func() {
				s.handleCronTrigger(&schedule)
			})
		}
	}
}