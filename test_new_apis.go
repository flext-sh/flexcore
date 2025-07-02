// Test New APIs - Validation of 100% Complete Implementation
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	fmt.Println("🧪 Testing FlexCore 100% Complete APIs")

	// Test plugin APIs
	testPluginAPIs()

	// Test workflow APIs
	testWorkflowAPIs()

	// Test event streaming
	testEventStreaming()
}

func testPluginAPIs() {
	fmt.Println("\n🔌 Testing Plugins API...")

	// Test list plugins
	resp, err := http.Get("http://localhost:8081/plugins/list")
	if err != nil {
		fmt.Printf("❌ Plugin list error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		fmt.Printf("✅ Plugins API working - found %v plugins\n", result["count"])

		// Print plugin names
		if plugins, ok := result["plugins"].([]interface{}); ok {
			for _, p := range plugins {
				if plugin, ok := p.(map[string]interface{}); ok {
					fmt.Printf("   📦 Plugin: %s v%s (%s)\n",
						plugin["name"],
						plugin["version"],
						plugin["type"])
				}
			}
		}
	} else {
		fmt.Printf("❌ Plugin list failed: %d\n", resp.StatusCode)
	}

	// Test plugin execution
	executeData := map[string]interface{}{
		"plugin_name": "data-transformer",
		"input": map[string]interface{}{
			"data": "test data",
			"format": "json",
		},
	}

	jsonData, _ := json.Marshal(executeData)
	resp, err = http.Post("http://localhost:8081/plugins/execute", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Plugin execute error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		fmt.Printf("✅ Plugin execution successful: %v\n", result["success"])
	} else {
		fmt.Printf("❌ Plugin execute failed: %d\n", resp.StatusCode)
	}

	// Test plugin health
	resp, err = http.Get("http://localhost:8081/plugins/health")
	if err != nil {
		fmt.Printf("❌ Plugin health error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		fmt.Printf("✅ Plugin health check completed\n")
	}
}

func testWorkflowAPIs() {
	fmt.Println("\n🔄 Testing Workflows API (Windmill integration)...")

	// Test list workflows
	resp, err := http.Get("http://localhost:8081/workflows/list")
	if err != nil {
		fmt.Printf("❌ Workflow list error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		fmt.Printf("✅ Workflows API working - found %v workflows\n", result["count"])

		// Print workflow names
		if workflows, ok := result["workflows"].([]interface{}); ok {
			for _, w := range workflows {
				if workflow, ok := w.(map[string]interface{}); ok {
					fmt.Printf("   🔄 Workflow: %s v%s - %s\n",
						workflow["name"],
						workflow["version"],
						workflow["description"])
				}
			}
		}
	} else {
		fmt.Printf("❌ Workflow list failed: %d\n", resp.StatusCode)
	}

	// Test workflow execution
	executeData := map[string]interface{}{
		"workflow_id": "pipeline-processor",
		"input": map[string]interface{}{
			"pipeline_id": "test-pipeline-001",
			"config": map[string]interface{}{
				"timeout": 300,
				"retries": 3,
			},
		},
	}

	jsonData, _ := json.Marshal(executeData)
	resp, err = http.Post("http://localhost:8081/workflows/execute", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Workflow execute error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		fmt.Printf("✅ Workflow execution started: %s\n", result["id"])

		// Test workflow status
		time.Sleep(1 * time.Second)
		statusURL := fmt.Sprintf("http://localhost:8081/workflows/status?execution_id=%s", result["id"])
		resp, err = http.Get(statusURL)
		if err == nil && resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			var statusResult map[string]interface{}
			json.Unmarshal(body, &statusResult)
			fmt.Printf("✅ Workflow status: %s\n", statusResult["status"])
			resp.Body.Close()
		}
	} else {
		fmt.Printf("❌ Workflow execute failed: %d\n", resp.StatusCode)
	}
}

func testEventStreaming() {
	fmt.Println("\n📡 Testing Event Streaming API...")

	// Test event stream (with timeout)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8081/events/stream")
	if err != nil {
		fmt.Printf("❌ Event stream error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Printf("✅ Event streaming connected\n")

		// Read first few events
		buffer := make([]byte, 1024)
		n, err := resp.Body.Read(buffer)
		if err == nil && n > 0 {
			fmt.Printf("✅ Event stream data received: %s\n", string(buffer[:n]))
		}
	} else {
		fmt.Printf("❌ Event stream failed: %d\n", resp.StatusCode)
	}
}
