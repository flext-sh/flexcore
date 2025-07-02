// FlexCore REAL Performance & Load Testing - 100% Production Validation
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Performance Test Configuration
type LoadTestConfig struct {
	BaseURL           string
	ConcurrentUsers   int
	RequestsPerUser   int
	TestDuration      time.Duration
	RampUpPeriod      time.Duration
	Services          []ServiceTest
}

type ServiceTest struct {
	Name     string
	Endpoint string
	Method   string
	Payload  interface{}
	Weight   int // Relative frequency
}

type PerformanceMetrics struct {
	ServiceName        string
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalResponseTime  int64
	MinResponseTime    int64
	MaxResponseTime    int64
	ResponseTimes      []int64
	ErrorRate          float64
	ThroughputRPS      float64
	P50ResponseTime    int64
	P95ResponseTime    int64
	P99ResponseTime    int64
}

type LoadTestResult struct {
	TestDuration       time.Duration
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	OverallThroughput  float64
	OverallErrorRate   float64
	ServiceMetrics     map[string]*PerformanceMetrics
	SystemHealth       map[string]interface{}
	Recommendations    []string
}

type LoadTester struct {
	config   LoadTestConfig
	results  map[string]*PerformanceMetrics
	mu       sync.RWMutex
	client   *http.Client
	stopChan chan bool
}

func NewLoadTester() *LoadTester {
	config := LoadTestConfig{
		BaseURL:         "http://localhost",
		ConcurrentUsers: 50,
		RequestsPerUser: 100,
		TestDuration:    5 * time.Minute,
		RampUpPeriod:    30 * time.Second,
		Services: []ServiceTest{
			{
				Name:     "Event Store",
				Endpoint: ":8095/events/append",
				Method:   "POST",
				Weight:   30,
				Payload: map[string]interface{}{
					"aggregate_id":   "load-test-{{ID}}",
					"aggregate_type": "LoadTest",
					"event_type":     "LoadTestEvent",
					"event_data": map[string]interface{}{
						"test_type": "performance",
						"timestamp": time.Now().Format(time.RFC3339),
						"data":      "performance test data payload",
					},
					"metadata": map[string]interface{}{
						"test": "load_test",
						"user": "performance_tester",
					},
				},
			},
			{
				Name:     "API Gateway CQRS Commands",
				Endpoint: ":8100/cqrs/commands",
				Method:   "POST",
				Weight:   25,
				Payload: map[string]interface{}{
					"command_id":   "load-cmd-{{ID}}",
					"command_type": "ProcessLoadTest",
					"aggregate_id": "load-test-{{ID}}",
					"payload": map[string]interface{}{
						"test_data": "load test command payload",
						"intensity": "high",
					},
				},
			},
			{
				Name:     "Data Processor",
				Endpoint: ":8097/process",
				Method:   "POST",
				Weight:   20,
				Payload: map[string]interface{}{
					"data":         `{"name": "Load Test", "value": 123, "type": "performance", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`,
					"input_format": "json",
					"output_format": "json",
					"metadata": map[string]string{
						"test": "load_test",
					},
				},
			},
			{
				Name:     "Script Engine",
				Endpoint: ":8098/execute",
				Method:   "POST",
				Weight:   15,
				Payload: map[string]interface{}{
					"script_id":   "load-script-{{ID}}",
					"script_type": "python",
					"code":        "import json\nresult = {'load_test': True, 'result': 'success'}\nprint(json.dumps(result))\nreturn result",
					"arguments": map[string]interface{}{
						"test_type": "load_test",
					},
					"timeout": 10,
				},
			},
			{
				Name:     "Read Model Projector",
				Endpoint: ":8099/project",
				Method:   "POST",
				Weight:   10,
				Payload: map[string]interface{}{
					"aggregate_id":   "load-test-{{ID}}",
					"aggregate_type": "LoadTest",
					"model_type":     "performance_metrics",
					"from_version":   0,
				},
			},
		},
	}

	return &LoadTester{
		config: config,
		results: make(map[string]*PerformanceMetrics),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan bool, 1),
	}
}

func (lt *LoadTester) RunLoadTest() (*LoadTestResult, error) {
	fmt.Println("🚀 Starting FlexCore REAL Performance & Load Test")
	fmt.Println("================================================")
	fmt.Printf("📊 Configuration:\n")
	fmt.Printf("   Concurrent Users: %d\n", lt.config.ConcurrentUsers)
	fmt.Printf("   Requests per User: %d\n", lt.config.RequestsPerUser)
	fmt.Printf("   Test Duration: %v\n", lt.config.TestDuration)
	fmt.Printf("   Ramp-up Period: %v\n", lt.config.RampUpPeriod)
	fmt.Printf("   Services Under Test: %d\n", len(lt.config.Services))
	fmt.Println()

	// Initialize metrics for each service
	for _, service := range lt.config.Services {
		lt.results[service.Name] = &PerformanceMetrics{
			ServiceName:    service.Name,
			MinResponseTime: math.MaxInt64,
			ResponseTimes:  []int64{},
		}
	}

	startTime := time.Now()

	// Start load generation
	var wg sync.WaitGroup
	
	// Ramp-up users gradually
	userInterval := lt.config.RampUpPeriod / time.Duration(lt.config.ConcurrentUsers)
	
	for i := 0; i < lt.config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			
			// Stagger user start times
			time.Sleep(time.Duration(userID) * userInterval)
			lt.runUserSession(userID)
		}(i)
	}

	// Stop test after duration
	go func() {
		time.Sleep(lt.config.TestDuration)
		close(lt.stopChan)
	}()

	wg.Wait()
	
	testDuration := time.Since(startTime)

	// Calculate final results
	return lt.calculateResults(testDuration), nil
}

func (lt *LoadTester) runUserSession(userID int) {
	requestCount := 0
	
	for {
		select {
		case <-lt.stopChan:
			return
		default:
			if requestCount >= lt.config.RequestsPerUser {
				return
			}
			
			// Select service based on weight
			service := lt.selectServiceByWeight()
			lt.executeRequest(userID, requestCount, service)
			requestCount++
			
			// Small delay between requests
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (lt *LoadTester) selectServiceByWeight() ServiceTest {
	totalWeight := 0
	for _, service := range lt.config.Services {
		totalWeight += service.Weight
	}
	
	random := time.Now().UnixNano() % int64(totalWeight)
	current := int64(0)
	
	for _, service := range lt.config.Services {
		current += int64(service.Weight)
		if random < current {
			return service
		}
	}
	
	return lt.config.Services[0] // Fallback
}

func (lt *LoadTester) executeRequest(userID, requestID int, service ServiceTest) {
	startTime := time.Now()
	
	// Prepare payload with dynamic values
	payload := lt.preparePayload(service.Payload, userID, requestID)
	payloadJSON, _ := json.Marshal(payload)
	
	// Create request
	url := lt.config.BaseURL + service.Endpoint
	req, err := http.NewRequest(service.Method, url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		lt.recordError(service.Name, time.Since(startTime).Milliseconds())
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Load-Test", "true")
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	req.Header.Set("X-Request-ID", fmt.Sprintf("%d", requestID))
	
	// Execute request
	resp, err := lt.client.Do(req)
	responseTime := time.Since(startTime).Milliseconds()
	
	if err != nil {
		lt.recordError(service.Name, responseTime)
		return
	}
	defer resp.Body.Close()
	
	// Record metrics
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		lt.recordSuccess(service.Name, responseTime)
	} else {
		lt.recordError(service.Name, responseTime)
	}
}

func (lt *LoadTester) preparePayload(payload interface{}, userID, requestID int) interface{} {
	// Convert to string, replace placeholders, then back to interface
	jsonBytes, _ := json.Marshal(payload)
	jsonStr := string(jsonBytes)
	
	// Replace placeholders
	id := fmt.Sprintf("%d-%d", userID, requestID)
	jsonStr = fmt.Sprintf(jsonStr, map[string]string{
		"{{ID}}": id,
	})
	
	var result interface{}
	json.Unmarshal([]byte(jsonStr), &result)
	return result
}

func (lt *LoadTester) recordSuccess(serviceName string, responseTime int64) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	metrics := lt.results[serviceName]
	atomic.AddInt64(&metrics.TotalRequests, 1)
	atomic.AddInt64(&metrics.SuccessfulRequests, 1)
	atomic.AddInt64(&metrics.TotalResponseTime, responseTime)
	
	if responseTime < metrics.MinResponseTime {
		metrics.MinResponseTime = responseTime
	}
	if responseTime > metrics.MaxResponseTime {
		metrics.MaxResponseTime = responseTime
	}
	
	metrics.ResponseTimes = append(metrics.ResponseTimes, responseTime)
}

func (lt *LoadTester) recordError(serviceName string, responseTime int64) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	metrics := lt.results[serviceName]
	atomic.AddInt64(&metrics.TotalRequests, 1)
	atomic.AddInt64(&metrics.FailedRequests, 1)
	atomic.AddInt64(&metrics.TotalResponseTime, responseTime)
	
	metrics.ResponseTimes = append(metrics.ResponseTimes, responseTime)
}

func (lt *LoadTester) calculateResults(testDuration time.Duration) *LoadTestResult {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	
	result := &LoadTestResult{
		TestDuration:   testDuration,
		ServiceMetrics: make(map[string]*PerformanceMetrics),
		Recommendations: []string{},
	}
	
	var totalRequests, totalSuccessful, totalFailed int64
	
	for serviceName, metrics := range lt.results {
		// Calculate percentiles
		lt.calculatePercentiles(metrics)
		
		// Calculate rates
		metrics.ErrorRate = float64(metrics.FailedRequests) / float64(metrics.TotalRequests) * 100
		metrics.ThroughputRPS = float64(metrics.TotalRequests) / testDuration.Seconds()
		
		result.ServiceMetrics[serviceName] = metrics
		
		totalRequests += metrics.TotalRequests
		totalSuccessful += metrics.SuccessfulRequests
		totalFailed += metrics.FailedRequests
	}
	
	result.TotalRequests = totalRequests
	result.SuccessfulRequests = totalSuccessful
	result.FailedRequests = totalFailed
	result.OverallThroughput = float64(totalRequests) / testDuration.Seconds()
	result.OverallErrorRate = float64(totalFailed) / float64(totalRequests) * 100
	
	// Generate recommendations
	result.Recommendations = lt.generateRecommendations(result)
	
	// Check system health
	result.SystemHealth = lt.checkSystemHealth()
	
	return result
}

func (lt *LoadTester) calculatePercentiles(metrics *PerformanceMetrics) {
	if len(metrics.ResponseTimes) == 0 {
		return
	}
	
	// Sort response times
	times := make([]int64, len(metrics.ResponseTimes))
	copy(times, metrics.ResponseTimes)
	
	for i := 0; i < len(times); i++ {
		for j := i + 1; j < len(times); j++ {
			if times[i] > times[j] {
				times[i], times[j] = times[j], times[i]
			}
		}
	}
	
	n := len(times)
	metrics.P50ResponseTime = times[n*50/100]
	metrics.P95ResponseTime = times[n*95/100]
	metrics.P99ResponseTime = times[n*99/100]
}

func (lt *LoadTester) generateRecommendations(result *LoadTestResult) []string {
	recommendations := []string{}
	
	// Overall performance analysis
	if result.OverallErrorRate > 5.0 {
		recommendations = append(recommendations, "🚨 HIGH ERROR RATE: Consider scaling services or optimizing error handling")
	}
	
	if result.OverallThroughput < 100 {
		recommendations = append(recommendations, "⚡ LOW THROUGHPUT: Consider horizontal scaling or performance optimization")
	}
	
	// Service-specific analysis
	for serviceName, metrics := range result.ServiceMetrics {
		if metrics.ErrorRate > 10.0 {
			recommendations = append(recommendations, fmt.Sprintf("🔧 %s: High error rate (%.1f%%) - investigate service stability", serviceName, metrics.ErrorRate))
		}
		
		if metrics.P95ResponseTime > 5000 {
			recommendations = append(recommendations, fmt.Sprintf("⏱️ %s: High P95 response time (%dms) - optimize performance", serviceName, metrics.P95ResponseTime))
		}
		
		if metrics.ThroughputRPS < 10 {
			recommendations = append(recommendations, fmt.Sprintf("📈 %s: Low throughput (%.1f RPS) - consider scaling", serviceName, metrics.ThroughputRPS))
		}
	}
	
	// Positive feedback
	if result.OverallErrorRate < 1.0 && result.OverallThroughput > 200 {
		recommendations = append(recommendations, "✅ EXCELLENT PERFORMANCE: System demonstrates production-ready stability and throughput")
	}
	
	return recommendations
}

func (lt *LoadTester) checkSystemHealth() map[string]interface{} {
	health := map[string]interface{}{
		"event_store_healthy":    lt.checkServiceHealth(":8095/health"),
		"api_gateway_healthy":    lt.checkServiceHealth(":8100/system/health"),
		"data_processor_healthy": lt.checkServiceHealth(":8097/health"),
		"script_engine_healthy":  lt.checkServiceHealth(":8098/health"),
		"read_model_healthy":     lt.checkServiceHealth(":8099/health"),
		"overall_status":         "unknown",
	}
	
	healthyCount := 0
	totalServices := 5
	
	for key, value := range health {
		if key != "overall_status" {
			if healthy, ok := value.(bool); ok && healthy {
				healthyCount++
			}
		}
	}
	
	if healthyCount == totalServices {
		health["overall_status"] = "healthy"
	} else if healthyCount >= totalServices*80/100 {
		health["overall_status"] = "degraded"
	} else {
		health["overall_status"] = "unhealthy"
	}
	
	health["healthy_services"] = healthyCount
	health["total_services"] = totalServices
	
	return health
}

func (lt *LoadTester) checkServiceHealth(endpoint string) bool {
	url := lt.config.BaseURL + endpoint
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == 200
}

func (lt *LoadTester) PrintResults(result *LoadTestResult) {
	fmt.Println("\n🎯 FLEXCORE PERFORMANCE TEST RESULTS")
	fmt.Println("=====================================")
	
	fmt.Printf("📊 Overall Performance:\n")
	fmt.Printf("   Test Duration: %v\n", result.TestDuration)
	fmt.Printf("   Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("   Successful: %d (%.1f%%)\n", result.SuccessfulRequests, 
		float64(result.SuccessfulRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("   Failed: %d (%.1f%%)\n", result.FailedRequests, result.OverallErrorRate)
	fmt.Printf("   Overall Throughput: %.1f RPS\n", result.OverallThroughput)
	fmt.Println()
	
	fmt.Printf("🔧 Service Performance Details:\n")
	for serviceName, metrics := range result.ServiceMetrics {
		fmt.Printf("   %s:\n", serviceName)
		fmt.Printf("      Requests: %d | Success: %d | Failed: %d\n", 
			metrics.TotalRequests, metrics.SuccessfulRequests, metrics.FailedRequests)
		fmt.Printf("      Throughput: %.1f RPS | Error Rate: %.1f%%\n", 
			metrics.ThroughputRPS, metrics.ErrorRate)
		fmt.Printf("      Response Times: Min=%dms | P50=%dms | P95=%dms | P99=%dms | Max=%dms\n",
			metrics.MinResponseTime, metrics.P50ResponseTime, metrics.P95ResponseTime,
			metrics.P99ResponseTime, metrics.MaxResponseTime)
		fmt.Println()
	}
	
	fmt.Printf("🏥 System Health Status:\n")
	if status, ok := result.SystemHealth["overall_status"].(string); ok {
		fmt.Printf("   Overall Status: %s\n", status)
	}
	if healthy, ok := result.SystemHealth["healthy_services"].(int); ok {
		if total, ok := result.SystemHealth["total_services"].(int); ok {
			fmt.Printf("   Services Healthy: %d/%d\n", healthy, total)
		}
	}
	fmt.Println()
	
	fmt.Printf("💡 Performance Recommendations:\n")
	if len(result.Recommendations) == 0 {
		fmt.Printf("   ✅ No issues detected - system performing optimally!\n")
	} else {
		for _, rec := range result.Recommendations {
			fmt.Printf("   %s\n", rec)
		}
	}
	fmt.Println()
	
	// Performance classification
	lt.classifyPerformance(result)
}

func (lt *LoadTester) classifyPerformance(result *LoadTestResult) {
	fmt.Printf("🏆 PERFORMANCE CLASSIFICATION:\n")
	
	score := 0
	maxScore := 5
	
	// Throughput score
	if result.OverallThroughput > 500 {
		score++
		fmt.Printf("   ✅ Throughput: EXCELLENT (>500 RPS)\n")
	} else if result.OverallThroughput > 200 {
		score++
		fmt.Printf("   ✅ Throughput: GOOD (>200 RPS)\n")
	} else if result.OverallThroughput > 100 {
		fmt.Printf("   ⚠️ Throughput: ACCEPTABLE (>100 RPS)\n")
	} else {
		fmt.Printf("   ❌ Throughput: POOR (<100 RPS)\n")
	}
	
	// Error rate score
	if result.OverallErrorRate < 1.0 {
		score++
		fmt.Printf("   ✅ Error Rate: EXCELLENT (<1%%)\n")
	} else if result.OverallErrorRate < 5.0 {
		score++
		fmt.Printf("   ✅ Error Rate: GOOD (<5%%)\n")
	} else if result.OverallErrorRate < 10.0 {
		fmt.Printf("   ⚠️ Error Rate: ACCEPTABLE (<10%%)\n")
	} else {
		fmt.Printf("   ❌ Error Rate: POOR (>10%%)\n")
	}
	
	// Response time score (check P95 across services)
	avgP95 := int64(0)
	serviceCount := int64(len(result.ServiceMetrics))
	for _, metrics := range result.ServiceMetrics {
		avgP95 += metrics.P95ResponseTime
	}
	if serviceCount > 0 {
		avgP95 /= serviceCount
	}
	
	if avgP95 < 1000 {
		score++
		fmt.Printf("   ✅ Response Time: EXCELLENT (P95 <1s)\n")
	} else if avgP95 < 3000 {
		score++
		fmt.Printf("   ✅ Response Time: GOOD (P95 <3s)\n")
	} else if avgP95 < 5000 {
		fmt.Printf("   ⚠️ Response Time: ACCEPTABLE (P95 <5s)\n")
	} else {
		fmt.Printf("   ❌ Response Time: POOR (P95 >5s)\n")
	}
	
	// System health score
	if status, ok := result.SystemHealth["overall_status"].(string); ok {
		if status == "healthy" {
			score++
			fmt.Printf("   ✅ System Health: EXCELLENT (All services healthy)\n")
		} else if status == "degraded" {
			fmt.Printf("   ⚠️ System Health: DEGRADED (Some services unhealthy)\n")
		} else {
			fmt.Printf("   ❌ System Health: POOR (Multiple services down)\n")
		}
	}
	
	// Stability score (based on consistency)
	stabilityScore := 0
	for _, metrics := range result.ServiceMetrics {
		if metrics.ErrorRate < 5.0 && metrics.ThroughputRPS > 10 {
			stabilityScore++
		}
	}
	
	if stabilityScore == len(result.ServiceMetrics) {
		score++
		fmt.Printf("   ✅ Stability: EXCELLENT (All services stable)\n")
	} else if stabilityScore >= len(result.ServiceMetrics)*80/100 {
		fmt.Printf("   ⚠️ Stability: GOOD (Most services stable)\n")
	} else {
		fmt.Printf("   ❌ Stability: POOR (Multiple unstable services)\n")
	}
	
	// Final classification
	fmt.Println()
	percentage := float64(score) / float64(maxScore) * 100
	
	fmt.Printf("🎖️ FINAL PERFORMANCE GRADE: %.0f%% (%d/%d)\n", percentage, score, maxScore)
	
	if percentage >= 90 {
		fmt.Printf("🏆 CLASSIFICATION: PRODUCTION-READY EXCELLENCE\n")
		fmt.Printf("✅ System ready for enterprise production deployment!\n")
	} else if percentage >= 70 {
		fmt.Printf("🥈 CLASSIFICATION: PRODUCTION-READY WITH OPTIMIZATIONS\n")
		fmt.Printf("⚡ System ready for production with minor optimizations\n")
	} else if percentage >= 50 {
		fmt.Printf("🥉 CLASSIFICATION: REQUIRES OPTIMIZATION\n")
		fmt.Printf("🔧 System needs performance improvements before production\n")
	} else {
		fmt.Printf("❌ CLASSIFICATION: NOT PRODUCTION-READY\n")
		fmt.Printf("🚨 System requires significant improvements\n")
	}
}

func main() {
	fmt.Println("🚀 FlexCore REAL Performance & Load Testing Suite")
	fmt.Println("===============================================")
	fmt.Println("⚡ Testing ALL FlexCore services under load")
	fmt.Println("📊 Measuring throughput, latency, and stability")
	fmt.Println("🎯 Validating production-readiness")
	fmt.Println()

	tester := NewLoadTester()
	
	// Wait for user confirmation
	fmt.Print("Press ENTER to start the load test (this will generate significant traffic)...")
	fmt.Scanln()
	
	result, err := tester.RunLoadTest()
	if err != nil {
		log.Fatal("Load test failed:", err)
	}
	
	tester.PrintResults(result)
	
	fmt.Println("\n🎯 LOAD TEST COMPLETED")
	fmt.Printf("⏰ Completed at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("📋 READY FOR FINAL 100%% VALIDATION\n")
}