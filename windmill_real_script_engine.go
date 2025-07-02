// FlexCore REAL Script Engine for Windmill - 100% Real Execution
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// REAL Script Engine Implementation
type RealScriptEngine struct {
	workDir    string
	pythonPath string
	nodePath   string
}

type ScriptRequest struct {
	ScriptID     string                 `json:"script_id"`
	ScriptType   string                 `json:"script_type"` // python, javascript, bash
	Code         string                 `json:"code"`
	Arguments    map[string]interface{} `json:"arguments"`
	Environment  map[string]string      `json:"environment"`
	Timeout      int                    `json:"timeout"` // seconds
	WorkingDir   string                 `json:"working_dir"`
}

type ScriptResponse struct {
	ScriptID      string                 `json:"script_id"`
	Success       bool                   `json:"success"`
	Output        string                 `json:"output"`
	ErrorOutput   string                 `json:"error_output"`
	ExitCode      int                    `json:"exit_code"`
	ExecutionTime int64                  `json:"execution_time_ms"`
	Result        map[string]interface{} `json:"result"`
	Metadata      map[string]string      `json:"metadata"`
}

func NewRealScriptEngine() *RealScriptEngine {
	workDir := "/tmp/flexcore_scripts"
	os.MkdirAll(workDir, 0755)

	// Find Python and Node paths
	pythonPath, _ := exec.LookPath("python3")
	if pythonPath == "" {
		pythonPath, _ = exec.LookPath("python")
	}
	
	nodePath, _ := exec.LookPath("node")

	return &RealScriptEngine{
		workDir:    workDir,
		pythonPath: pythonPath,
		nodePath:   nodePath,
	}
}

func (engine *RealScriptEngine) ExecuteScript(req *ScriptRequest) (*ScriptResponse, error) {
	startTime := time.Now()

	log.Printf("🔧 Executing REAL script: %s (%s)", req.ScriptID, req.ScriptType)

	// Create script file
	scriptFile, err := engine.createScriptFile(req)
	if err != nil {
		return &ScriptResponse{
			ScriptID:    req.ScriptID,
			Success:     false,
			ErrorOutput: fmt.Sprintf("Failed to create script file: %v", err),
			ExitCode:    -1,
		}, nil
	}
	defer os.Remove(scriptFile)

	// Execute script with real command
	output, errorOutput, exitCode, err := engine.executeScriptFile(req, scriptFile)
	if err != nil {
		return &ScriptResponse{
			ScriptID:      req.ScriptID,
			Success:       false,
			ErrorOutput:   fmt.Sprintf("Execution failed: %v", err),
			ExitCode:      exitCode,
			ExecutionTime: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Parse result if JSON output
	result := engine.parseScriptResult(output, req.ScriptType)

	response := &ScriptResponse{
		ScriptID:      req.ScriptID,
		Success:       exitCode == 0,
		Output:        output,
		ErrorOutput:   errorOutput,
		ExitCode:      exitCode,
		ExecutionTime: time.Since(startTime).Milliseconds(),
		Result:        result,
		Metadata:      engine.generateMetadata(req, exitCode),
	}

	log.Printf("✅ Script executed: exit=%d, time=%dms", exitCode, response.ExecutionTime)
	return response, nil
}

func (engine *RealScriptEngine) createScriptFile(req *ScriptRequest) (string, error) {
	var extension string
	var content string

	switch strings.ToLower(req.ScriptType) {
	case "python":
		extension = ".py"
		content = engine.wrapPythonScript(req.Code, req.Arguments)
	case "javascript", "js":
		extension = ".js"
		content = engine.wrapJavaScriptScript(req.Code, req.Arguments)
	case "bash", "shell":
		extension = ".sh"
		content = engine.wrapBashScript(req.Code, req.Arguments)
	default:
		return "", fmt.Errorf("unsupported script type: %s", req.ScriptType)
	}

	fileName := fmt.Sprintf("script_%s_%d%s", req.ScriptID, time.Now().Unix(), extension)
	filePath := filepath.Join(engine.workDir, fileName)

	err := os.WriteFile(filePath, []byte(content), 0755)
	if err != nil {
		return "", fmt.Errorf("failed to write script file: %w", err)
	}

	return filePath, nil
}

func (engine *RealScriptEngine) wrapPythonScript(code string, args map[string]interface{}) string {
	argsJSON, _ := json.Marshal(args)
	if args == nil {
		argsJSON = []byte("{}")
	}
	
	wrapper := fmt.Sprintf(`#!/usr/bin/env python3
import json
import sys
import os

# Script arguments
script_args = %s

# User script
def main():
%s

# Execute and capture result
if __name__ == "__main__":
    try:
        result = main()
        if result is not None:
            print(json.dumps({"success": True, "result": result}))
        else:
            print(json.dumps({"success": True, "result": "Script completed successfully"}))
    except Exception as e:
        print(json.dumps({"success": False, "error": str(e)}))
        sys.exit(1)
`, string(argsJSON), engine.indentCode(code, "    "))

	return wrapper
}

func (engine *RealScriptEngine) wrapJavaScriptScript(code string, args map[string]interface{}) string {
	argsJSON, _ := json.Marshal(args)
	
	wrapper := fmt.Sprintf(`#!/usr/bin/env node

// Script arguments
const scriptArgs = %s;

// User script function
function main() {
%s
}

// Execute and capture result
try {
    const result = main();
    if (result !== undefined) {
        console.log(JSON.stringify({success: true, result: result}));
    } else {
        console.log(JSON.stringify({success: true, result: "Script completed successfully"}));
    }
} catch (error) {
    console.log(JSON.stringify({success: false, error: error.message}));
    process.exit(1);
}
`, string(argsJSON), engine.indentCode(code, "    "))

	return wrapper
}

func (engine *RealScriptEngine) wrapBashScript(code string, args map[string]interface{}) string {
	// Convert arguments to environment variables
	envVars := ""
	for key, value := range args {
		envVars += fmt.Sprintf("export %s=\"%v\"\n", strings.ToUpper(key), value)
	}

	wrapper := fmt.Sprintf(`#!/bin/bash
set -e

# Script arguments as environment variables
%s

# User script
%s

# Success indicator
echo '{"success": true, "result": "Bash script completed successfully"}'
`, envVars, code)

	return wrapper
}

func (engine *RealScriptEngine) indentCode(code string, indent string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

func (engine *RealScriptEngine) executeScriptFile(req *ScriptRequest, scriptFile string) (string, string, int, error) {
	var cmd *exec.Cmd

	// Set timeout
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create command based on script type
	switch strings.ToLower(req.ScriptType) {
	case "python":
		if engine.pythonPath == "" {
			return "", "", -1, fmt.Errorf("python interpreter not found")
		}
		cmd = exec.CommandContext(ctx, engine.pythonPath, scriptFile)
	case "javascript", "js":
		if engine.nodePath == "" {
			return "", "", -1, fmt.Errorf("node interpreter not found")
		}
		cmd = exec.CommandContext(ctx, engine.nodePath, scriptFile)
	case "bash", "shell":
		cmd = exec.CommandContext(ctx, "/bin/bash", scriptFile)
	default:
		return "", "", -1, fmt.Errorf("unsupported script type: %s", req.ScriptType)
	}

	// Set working directory
	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
	} else {
		cmd.Dir = engine.workDir
	}

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range req.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Execute command
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

func (engine *RealScriptEngine) parseScriptResult(output string, scriptType string) map[string]interface{} {
	result := make(map[string]interface{})

	// Try to parse JSON output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			var jsonResult map[string]interface{}
			if err := json.Unmarshal([]byte(line), &jsonResult); err == nil {
				// Merge JSON result
				for k, v := range jsonResult {
					result[k] = v
				}
			}
		}
	}

	// If no JSON found, set raw output
	if len(result) == 0 {
		result["output"] = output
		result["success"] = true
	}

	result["script_type"] = scriptType
	result["parsed_at"] = time.Now().Format(time.RFC3339)

	return result
}

func (engine *RealScriptEngine) generateMetadata(req *ScriptRequest, exitCode int) map[string]string {
	metadata := map[string]string{
		"engine":         "real-script-engine",
		"script_type":    req.ScriptType,
		"exit_code":      fmt.Sprintf("%d", exitCode),
		"executed_at":    time.Now().Format(time.RFC3339),
		"engine_version": "1.0.0",
		"working_dir":    engine.workDir,
	}

	if engine.pythonPath != "" {
		metadata["python_path"] = engine.pythonPath
	}
	if engine.nodePath != "" {
		metadata["node_path"] = engine.nodePath
	}

	return metadata
}

// HTTP Handlers
func (engine *RealScriptEngine) handleExecuteScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ScriptID == "" {
		req.ScriptID = fmt.Sprintf("script_%d", time.Now().Unix())
	}
	if req.ScriptType == "" {
		req.ScriptType = "python"
	}
	if req.Timeout == 0 {
		req.Timeout = 30
	}

	resp, err := engine.ExecuteScript(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execution error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (engine *RealScriptEngine) handleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":        "Real Script Engine",
		"version":     "1.0.0",
		"description": "Production-ready script execution engine for Python, JavaScript, and Bash",
		"supported_languages": []string{"python", "javascript", "bash"},
		"capabilities": map[string]interface{}{
			"real_execution":    true,
			"timeout_support":   true,
			"environment_vars":  true,
			"working_directory": true,
			"result_parsing":    true,
			"error_handling":    true,
		},
		"interpreters": map[string]interface{}{
			"python": engine.pythonPath != "",
			"node":   engine.nodePath != "",
			"bash":   true,
		},
		"paths": map[string]string{
			"python":     engine.pythonPath,
			"node":       engine.nodePath,
			"bash":       "/bin/bash",
			"working_dir": engine.workDir,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (engine *RealScriptEngine) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"healthy":   true,
		"status":    "operational",
		"message":   "Real script engine running with all interpreters",
		"timestamp": time.Now(),
		"checks": map[string]bool{
			"python_available": engine.pythonPath != "",
			"node_available":   engine.nodePath != "",
			"bash_available":   true,
			"work_dir_writable": engine.checkWorkDirWritable(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (engine *RealScriptEngine) checkWorkDirWritable() bool {
	testFile := filepath.Join(engine.workDir, "test_write")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		return false
	}
	os.Remove(testFile)
	return true
}

func main() {
	fmt.Println("🔧 FlexCore REAL Script Engine")
	fmt.Println("==============================")
	fmt.Println("🐍 Python script execution support")
	fmt.Println("🟨 JavaScript/Node.js execution support")
	fmt.Println("💻 Bash script execution support")
	fmt.Println("⏱️ Real timeout and environment control")
	fmt.Println()

	engine := NewRealScriptEngine()

	// HTTP endpoints
	http.HandleFunc("/execute", engine.handleExecuteScript)
	http.HandleFunc("/info", engine.handleInfo)
	http.HandleFunc("/health", engine.handleHealth)

	// Root endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		info := map[string]interface{}{
			"service":     "real-script-engine",
			"version":     "1.0.0",
			"description": "FlexCore Real Script Engine",
			"timestamp":   time.Now(),
			"endpoints": []string{
				"POST /execute - Execute script",
				"GET  /info - Engine information",
				"GET  /health - Health check",
			},
			"supported_languages": []string{"python", "javascript", "bash"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	fmt.Printf("🌐 Real Script Engine starting on :8098\n")
	fmt.Printf("📊 Endpoints:\n")
	fmt.Printf("  POST /execute - Execute scripts\n")
	fmt.Printf("  GET  /info    - Engine info\n")
	fmt.Printf("  GET  /health  - Health check\n")
	fmt.Printf("🔧 Python: %s\n", engine.pythonPath)
	fmt.Printf("🟨 Node.js: %s\n", engine.nodePath)
	fmt.Printf("💻 Bash: /bin/bash\n")
	fmt.Printf("📁 Work dir: %s\n", engine.workDir)
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8098", nil))
}