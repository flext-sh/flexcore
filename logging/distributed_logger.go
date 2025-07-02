// FlexCore Distributed Logging - REAL ELK Stack Integration
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/sirupsen/logrus"
)

// Log Levels
type LogLevel string

const (
	LOG_DEBUG LogLevel = "DEBUG"
	LOG_INFO  LogLevel = "INFO"
	LOG_WARN  LogLevel = "WARN"
	LOG_ERROR LogLevel = "ERROR"
	LOG_FATAL LogLevel = "FATAL"
)

// Structured Log Entry
type LogEntry struct {
	Timestamp   time.Time              `json:"@timestamp"`
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	Node        string                 `json:"node"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Component   string                 `json:"component"`
	Function    string                 `json:"function"`
	File        string                 `json:"file"`
	Line        int                    `json:"line"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Environment string                 `json:"environment"`
	Version     string                 `json:"version"`
}

// Distributed Logger
type DistributedLogger struct {
	esClient    *elasticsearch.Client
	indexPrefix string
	service     string
	node        string
	environment string
	version     string
	buffer      []LogEntry
	bufferMu    sync.Mutex
	batchSize   int
	flushTimer  *time.Timer
	logrus      *logrus.Logger
}

func NewDistributedLogger(config LoggerConfig) (*DistributedLogger, error) {
	// Initialize Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: config.ElasticsearchURLs,
		Username:  config.Username,
		Password:  config.Password,
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Initialize logrus for local logging
	logrusLogger := logrus.New()
	logrusLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	// Set log level
	switch strings.ToUpper(config.LogLevel) {
	case "DEBUG":
		logrusLogger.SetLevel(logrus.DebugLevel)
	case "INFO":
		logrusLogger.SetLevel(logrus.InfoLevel)
	case "WARN":
		logrusLogger.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logrusLogger.SetLevel(logrus.ErrorLevel)
	default:
		logrusLogger.SetLevel(logrus.InfoLevel)
	}

	dl := &DistributedLogger{
		esClient:    esClient,
		indexPrefix: config.IndexPrefix,
		service:     config.Service,
		node:        config.Node,
		environment: config.Environment,
		version:     config.Version,
		buffer:      make([]LogEntry, 0, config.BatchSize),
		batchSize:   config.BatchSize,
		logrus:      logrusLogger,
	}

	// Start flush timer
	dl.flushTimer = time.NewTimer(time.Duration(config.FlushIntervalSeconds) * time.Second)
	go dl.flushLoop()

	return dl, nil
}

type LoggerConfig struct {
	ElasticsearchURLs    []string `json:"elasticsearch_urls"`
	Username             string   `json:"username"`
	Password             string   `json:"password"`
	IndexPrefix          string   `json:"index_prefix"`
	Service              string   `json:"service"`
	Node                 string   `json:"node"`
	Environment          string   `json:"environment"`
	Version              string   `json:"version"`
	LogLevel             string   `json:"log_level"`
	BatchSize            int      `json:"batch_size"`
	FlushIntervalSeconds int      `json:"flush_interval_seconds"`
}

func (dl *DistributedLogger) createLogEntry(level LogLevel, message string, fields map[string]interface{}) LogEntry {
	// Get caller information
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Extract function name
	pc, _, _, ok := runtime.Caller(3)
	function := "unknown"
	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			function = fn.Name()
		}
	}

	entry := LogEntry{
		Timestamp:   time.Now().UTC(),
		Level:       level,
		Message:     message,
		Service:     dl.service,
		Node:        dl.node,
		Component:   extractComponent(file),
		Function:    function,
		File:        file,
		Line:        line,
		Fields:      fields,
		Environment: dl.environment,
		Version:     dl.version,
	}

	// Add stack trace for errors
	if level == LOG_ERROR || level == LOG_FATAL {
		entry.StackTrace = getStackTrace()
	}

	return entry
}

func extractComponent(file string) string {
	parts := strings.Split(file, "/")
	if len(parts) > 1 {
		return parts[len(parts)-2] // Get directory name
	}
	return "unknown"
}

func getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (dl *DistributedLogger) log(level LogLevel, message string, fields map[string]interface{}) {
	entry := dl.createLogEntry(level, message, fields)

	// Log locally with logrus
	logFields := logrus.Fields{
		"service":   entry.Service,
		"node":      entry.Node,
		"component": entry.Component,
		"function":  entry.Function,
	}

	// Add custom fields
	for k, v := range fields {
		logFields[k] = v
	}

	switch level {
	case LOG_DEBUG:
		dl.logrus.WithFields(logFields).Debug(message)
	case LOG_INFO:
		dl.logrus.WithFields(logFields).Info(message)
	case LOG_WARN:
		dl.logrus.WithFields(logFields).Warn(message)
	case LOG_ERROR:
		dl.logrus.WithFields(logFields).Error(message)
	case LOG_FATAL:
		dl.logrus.WithFields(logFields).Fatal(message)
	}

	// Add to buffer for Elasticsearch
	dl.bufferMu.Lock()
	dl.buffer = append(dl.buffer, entry)
	shouldFlush := len(dl.buffer) >= dl.batchSize
	dl.bufferMu.Unlock()

	if shouldFlush {
		go dl.flush()
	}
}

func (dl *DistributedLogger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	dl.log(LOG_DEBUG, message, f)
}

func (dl *DistributedLogger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	dl.log(LOG_INFO, message, f)
}

func (dl *DistributedLogger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	dl.log(LOG_WARN, message, f)
}

func (dl *DistributedLogger) Error(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	dl.log(LOG_ERROR, message, f)
}

func (dl *DistributedLogger) Fatal(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	dl.log(LOG_FATAL, message, f)
}

// Contextual logging methods
func (dl *DistributedLogger) WithTraceID(traceID string) *ContextLogger {
	return &ContextLogger{dl: dl, traceID: traceID}
}

func (dl *DistributedLogger) WithRequestID(requestID string) *ContextLogger {
	return &ContextLogger{dl: dl, requestID: requestID}
}

func (dl *DistributedLogger) WithUserID(userID string) *ContextLogger {
	return &ContextLogger{dl: dl, userID: userID}
}

type ContextLogger struct {
	dl        *DistributedLogger
	traceID   string
	spanID    string
	userID    string
	requestID string
}

func (cl *ContextLogger) Info(message string, fields ...map[string]interface{}) {
	entry := cl.dl.createLogEntry(LOG_INFO, message, mergeFields(fields...))
	entry.TraceID = cl.traceID
	entry.SpanID = cl.spanID
	entry.UserID = cl.userID
	entry.RequestID = cl.requestID

	cl.dl.bufferMu.Lock()
	cl.dl.buffer = append(cl.dl.buffer, entry)
	shouldFlush := len(cl.dl.buffer) >= cl.dl.batchSize
	cl.dl.bufferMu.Unlock()

	if shouldFlush {
		go cl.dl.flush()
	}
}

func mergeFields(fieldMaps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, fields := range fieldMaps {
		for k, v := range fields {
			result[k] = v
		}
	}
	return result
}

func (dl *DistributedLogger) flush() {
	dl.bufferMu.Lock()
	if len(dl.buffer) == 0 {
		dl.bufferMu.Unlock()
		return
	}

	entriesToFlush := make([]LogEntry, len(dl.buffer))
	copy(entriesToFlush, dl.buffer)
	dl.buffer = dl.buffer[:0] // Clear buffer
	dl.bufferMu.Unlock()

	// Create index name with date
	indexName := fmt.Sprintf("%s-%s", dl.indexPrefix, time.Now().Format("2006.01.02"))

	// Prepare bulk request
	var buf bytes.Buffer
	for _, entry := range entriesToFlush {
		// Index action
		action := map[string]interface{}{
			"index": map[string]string{
				"_index": indexName,
			},
		}
		actionJSON, _ := json.Marshal(action)
		buf.Write(actionJSON)
		buf.WriteString("\n")

		// Document
		entryJSON, _ := json.Marshal(entry)
		buf.Write(entryJSON)
		buf.WriteString("\n")
	}

	// Send to Elasticsearch
	resp, err := dl.esClient.Bulk(
		bytes.NewReader(buf.Bytes()),
		dl.esClient.Bulk.WithContext(context.Background()),
	)

	if err != nil {
		log.Printf("Failed to send logs to Elasticsearch: %v", err)
		return
	}

	defer resp.Body.Close()

	if resp.IsError() {
		log.Printf("Elasticsearch bulk request failed: %s", resp.Status())
	}
}

func (dl *DistributedLogger) flushLoop() {
	for range dl.flushTimer.C {
		dl.flush()
		dl.flushTimer.Reset(30 * time.Second) // Reset timer
	}
}

// Log Server for receiving logs from other services
type LogServer struct {
	logger *DistributedLogger
	server *http.Server
}

func NewLogServer(logger *DistributedLogger, port string) *LogServer {
	mux := http.NewServeMux()

	ls := &LogServer{
		logger: logger,
		server: &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		},
	}

	// Log ingestion endpoint
	mux.HandleFunc("/logs/ingest", ls.ingestHandler)
	mux.HandleFunc("/logs/health", ls.healthHandler)
	mux.HandleFunc("/logs/search", ls.searchHandler)

	return ls
}

func (ls *LogServer) ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var entries []LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Add entries to buffer
	ls.logger.bufferMu.Lock()
	ls.logger.buffer = append(ls.logger.buffer, entries...)
	shouldFlush := len(ls.logger.buffer) >= ls.logger.batchSize
	ls.logger.bufferMu.Unlock()

	if shouldFlush {
		go ls.logger.flush()
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accepted": len(entries),
		"timestamp": time.Now(),
	})
}

func (ls *LogServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"service": "distributed-logger",
		"timestamp": time.Now(),
		"buffer_size": len(ls.logger.buffer),
	})
}

func (ls *LogServer) searchHandler(w http.ResponseWriter, r *http.Request) {
	// Simple search implementation
	query := r.URL.Query().Get("q")
	service := r.URL.Query().Get("service")
	level := r.URL.Query().Get("level")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query": query,
		"service": service,
		"level": level,
		"message": "Search functionality requires full Elasticsearch query implementation",
		"suggestion": "Use Kibana for advanced log searching and visualization",
	})
}

func (ls *LogServer) Start() error {
	log.Printf("📝 Starting Log Server on %s", ls.server.Addr)
	return ls.server.ListenAndServe()
}

func (ls *LogServer) Stop(ctx context.Context) error {
	return ls.server.Shutdown(ctx)
}

func main() {
	fmt.Println("📝 FlexCore Distributed Logging System")
	fmt.Println("=====================================")
	fmt.Println("🔍 Real ELK Stack integration")
	fmt.Println("📊 Structured logging with metadata")
	fmt.Println("🔗 Distributed tracing correlation")
	fmt.Println("⏱️ Batch processing and buffering")
	fmt.Println()

	// Load configuration
	config := LoggerConfig{
		ElasticsearchURLs:    []string{"http://localhost:9200"},
		IndexPrefix:          "flexcore-logs",
		Service:              "flexcore-logging",
		Node:                 "node-1",
		Environment:          "production",
		Version:              "1.0.0",
		LogLevel:             "INFO",
		BatchSize:            50,
		FlushIntervalSeconds: 30,
	}

	logger, err := NewDistributedLogger(config)
	if err != nil {
		log.Fatalf("Failed to create distributed logger: %v", err)
	}

	logServer := NewLogServer(logger, "8999")

	// Test logging
	logger.Info("Distributed logging system started", map[string]interface{}{
		"component": "main",
		"config": config,
	})

	logger.WithTraceID("trace-123").Info("Example traced log entry", map[string]interface{}{
		"operation": "startup",
		"duration_ms": 150,
	})

	if err := logServer.Start(); err != nil {
		log.Fatalf("Log server failed: %v", err)
	}
}
