// Real Plugin System Test
// This tests the actual plugin loading and execution
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/flext/flexcore/infrastructure/plugins"
)

func main() {
	fmt.Println("🧪 Testing REAL Plugin System...")
	
	// Build the CSV extractor plugin first
	fmt.Println("🔨 Building CSV extractor plugin...")
	cmd := exec.Command("go", "build", "-o", "./plugins/csv-extractor", "./examples/plugins/csv_extractor")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		log.Fatalf("❌ Failed to build plugin: %v", err)
	}
	fmt.Println("✅ Plugin built successfully")
	
	// Create plugin manager
	pluginManager := plugins.NewPluginManager("./plugins")
	
	// Register plugin types
	pluginManager.RegisterPluginType("plugin", &plugins.ExtractorGRPCPlugin{})
	
	fmt.Println("🔍 Scanning for plugins...")
	ctx := context.Background()
	
	// Scan for plugins
	scanResult := pluginManager.ScanPluginDirectory(ctx)
	if scanResult.IsFailure() {
		log.Fatalf("❌ Failed to scan plugins: %v", scanResult.Error())
	}
	
	pluginPaths := scanResult.Value()
	fmt.Printf("📦 Found %d plugin(s)\n", len(pluginPaths))
	
	if len(pluginPaths) == 0 {
		log.Fatal("❌ No plugins found")
	}
	
	// Load plugins
	fmt.Println("🚀 Loading plugins...")
	loadResult := pluginManager.LoadAllPlugins(ctx)
	if loadResult.IsFailure() {
		log.Fatalf("❌ Failed to load plugins: %v", loadResult.Error())
	}
	
	loadedCount := loadResult.Value()
	fmt.Printf("✅ Loaded %d plugin(s)\n", loadedCount)
	
	// List loaded plugins
	fmt.Println("📋 Available plugins:")
	loadedPlugins := pluginManager.ListPlugins()
	for _, plugin := range loadedPlugins {
		fmt.Printf("  🔌 %s v%s (%s) - Active: %t\n", 
			plugin.Name, plugin.Version, plugin.Type, plugin.IsActive)
	}
	
	if len(loadedPlugins) == 0 {
		log.Fatal("❌ No plugins loaded")
	}
	
	// Test plugin execution
	fmt.Println("⚡ Testing plugin execution...")
	
	// Create test CSV file
	testCSV := "./test-data.csv"
	csvContent := `name,age,city
John,30,New York
Jane,25,Los Angeles
Bob,35,Chicago`
	
	err := os.WriteFile(testCSV, []byte(csvContent), 0644)
	if err != nil {
		log.Fatalf("❌ Failed to create test CSV: %v", err)
	}
	defer os.Remove(testCSV)
	
	// Execute plugin
	plugin := loadedPlugins[0]
	execResult := pluginManager.ExecutePlugin(ctx, plugin.ID, testCSV)
	if execResult.IsFailure() {
		log.Fatalf("❌ Plugin execution failed: %v", execResult.Error())
	}
	
	result := execResult.Value()
	fmt.Printf("🎉 Plugin execution successful!\n")
	fmt.Printf("📊 Result type: %T\n", result)
	
	// Display result
	if data, ok := result.([]interface{}); ok {
		fmt.Printf("📈 Processed %d records\n", len(data))
		if len(data) > 0 {
			fmt.Printf("🔍 First record: %+v\n", data[0])
		}
	}
	
	// Shutdown
	fmt.Println("🛑 Shutting down plugin system...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := pluginManager.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ Warning during shutdown: %v", err)
	}
	
	fmt.Println("✅ REAL Plugin System Test Completed Successfully!")
	fmt.Println("🎯 Plugin loading, execution, and data processing all working!")
}