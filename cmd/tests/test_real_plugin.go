package main

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/flext/flexcore/infrastructure/plugins"
)

func main() {
	fmt.Println("🔌 Testing REAL HashiCorp Plugin System...")

	// Build plugin first
	fmt.Println("Building plugin...")
	cmd := exec.Command("go", "build", "-o", "plugins/plugin-data-extractor", "./examples/plugins/data-extractor")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("❌ Plugin build failed: %v\n", err)
		return
	}
	fmt.Println("✅ Plugin built successfully")

	// Create REAL plugin manager
	manager := plugins.NewRealPluginManager("plugins")

	// Start manager
	ctx := context.Background()
	err = manager.Start(ctx)
	if err != nil {
		fmt.Printf("❌ Manager start failed: %v\n", err)
		return
	}
	fmt.Println("✅ Plugin manager started")

	// Load REAL plugin
	fmt.Println("Loading REAL plugin...")
	result := manager.LoadPlugin(ctx, "data-extractor", "plugins/plugin-data-extractor")
	if result.IsFailure() {
		fmt.Printf("❌ Plugin load failed: %v\n", result.Error())
		return
	}

	plugin := result.Value()
	fmt.Printf("✅ Plugin loaded: %s v%s\n", plugin.Info.Name, plugin.Info.Version)

	// Execute REAL plugin
	fmt.Println("Executing REAL plugin...")
	input := map[string]interface{}{
		"source": "test-database",
		"limit":  100,
	}

	execResult := manager.ExecutePlugin(ctx, "data-extractor", input)
	if execResult.IsFailure() {
		fmt.Printf("❌ Plugin execution failed: %v\n", execResult.Error())
		return
	}

	output := execResult.Value()
	fmt.Printf("✅ Plugin executed successfully\n")
	fmt.Printf("Output: %+v\n", output)

	// Test plugin health
	fmt.Println("Testing plugin health...")
	pluginResult := manager.GetPlugin("data-extractor")
	if pluginResult.IsSuccess() {
		p := pluginResult.Value()
		fmt.Printf("✅ Plugin status: %s\n", p.Status)
		fmt.Printf("✅ Execution count: %d\n", p.ExecutionCount)
	}

	// Unload plugin
	fmt.Println("Unloading plugin...")
	err = manager.UnloadPlugin("data-extractor")
	if err != nil {
		fmt.Printf("❌ Plugin unload failed: %v\n", err)
	} else {
		fmt.Printf("✅ Plugin unloaded successfully\n")
	}

	// Stop manager
	manager.Stop()
	fmt.Println("✅ Plugin manager stopped")

	fmt.Println("\n🎯 PLUGIN REAL TESTS COMPLETED")
}