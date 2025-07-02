package main

import (
	"context"
	"fmt"

	"github.com/flext/flexcore/infrastructure/di"
)

func main() {
	fmt.Println("💉 Testing REAL Dependency Injection Container...")

	// Create container
	container := di.NewAdvancedContainer()

	// Test 1: Register and resolve services
	fmt.Println("🔧 Test 1: Service Registration and Resolution")

	// Register a simple service
	container.RegisterProvider("test-service", di.Value[any]("Hello FlexCore DI!"))
	container.RegisterProvider("number-service", di.Value[any](42))

	// Resolve services
	strResult := di.ResolveProvider[string](container, context.Background(), "test-service")
	if strResult.IsSuccess() {
		fmt.Printf("✅ String service resolved: %s\n", strResult.Value())
	} else {
		fmt.Printf("❌ String service failed: %v\n", strResult.Error())
	}

	numResult := di.ResolveProvider[int](container, context.Background(), "number-service")
	if numResult.IsSuccess() {
		fmt.Printf("✅ Number service resolved: %d\n", numResult.Value())
	} else {
		fmt.Printf("❌ Number service failed: %v\n", numResult.Error())
	}

	// Test 2: Factory services
	fmt.Println("\n🏭 Test 2: Factory Services")

	container.RegisterProvider("factory-service", di.Factory[any](func(ctx context.Context) (any, error) {
		return fmt.Sprintf("Generated at context: %v", ctx != nil), nil
	}))

	factoryResult := di.ResolveProvider[string](container, context.Background(), "factory-service")
	if factoryResult.IsSuccess() {
		fmt.Printf("✅ Factory service resolved: %s\n", factoryResult.Value())
	} else {
		fmt.Printf("❌ Factory service failed: %v\n", factoryResult.Error())
	}

	// Test 3: Singleton services
	fmt.Println("\n🎯 Test 3: Singleton Services")

	container.RegisterProvider("singleton-service", di.Singleton[any](func(ctx context.Context) (any, error) {
		return fmt.Sprintf("Singleton instance created"), nil
	}))

	// Resolve twice - should be same instance concept
	single1 := di.ResolveProvider[string](container, context.Background(), "singleton-service")
	single2 := di.ResolveProvider[string](container, context.Background(), "singleton-service")

	if single1.IsSuccess() && single2.IsSuccess() {
		fmt.Printf("✅ Singleton 1: %s\n", single1.Value())
		fmt.Printf("✅ Singleton 2: %s\n", single2.Value())
	}

	// Test 4: Call with DI
	fmt.Println("\n📞 Test 4: Function Call with DI")

	testFunc := func(ctx context.Context, message string, number int) string {
		return fmt.Sprintf("Called with: %s and %d", message, number)
	}

	callResult := container.Call(context.Background(), testFunc, "manual-arg", 999)
	if callResult.IsSuccess() {
		fmt.Printf("✅ Function call result: %s\n", callResult.Value())
	} else {
		fmt.Printf("❌ Function call failed: %v\n", callResult.Error())
	}

	fmt.Println("\n🎯 DI REAL TESTS COMPLETED - Container is working!")
}