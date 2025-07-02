package main

import (
	"context"
	"fmt"

	"github.com/flext/flexcore/infrastructure/di"
)

// Test service interfaces
type DatabaseService interface {
	Connect() string
}

type LoggerService interface {
	Log(msg string)
}

// Real implementations
type PostgresDB struct {
	ConnectionString string
}

func (p *PostgresDB) Connect() string {
	return "Connected to PostgreSQL: " + p.ConnectionString
}

type ConsoleLogger struct {
	Prefix string
}

func (c *ConsoleLogger) Log(msg string) {
	fmt.Printf("[%s] %s\n", c.Prefix, msg)
}

// Service that depends on others
type UserService struct {
	DB     DatabaseService `wire:""`
	Logger LoggerService   `wire:""`
}

func (u *UserService) CreateUser(name string) string {
	u.Logger.Log("Creating user: " + name)
	conn := u.DB.Connect()
	return fmt.Sprintf("User %s created via %s", name, conn)
}

func main() {
	fmt.Println("💉 Testing REAL Dependency Injection Container...")

	// Create container
	container := di.NewAdvancedContainer()

	// Register services
	fmt.Println("Registering services...")
	
	// Register singleton DB service
	container.RegisterProvider("database", di.Singleton[any](func(ctx context.Context) (any, error) {
		return &PostgresDB{ConnectionString: "postgres://localhost:5432/flexcore"}, nil
	}))

	// Register transient logger service
	container.RegisterProvider("logger", di.Factory[any](func(ctx context.Context) (any, error) {
		return &ConsoleLogger{Prefix: "FlexCore"}, nil
	}))

	fmt.Println("✅ Services registered")

	// Test 1: Direct resolution
	fmt.Println("\n🔍 Test 1: Direct Service Resolution")
	
	dbResult := di.ResolveProvider[DatabaseService](container, context.Background(), "database")
	if dbResult.IsSuccess() {
		db := dbResult.Value()
		result := db.Connect()
		fmt.Printf("✅ Database service: %s\n", result)
	} else {
		fmt.Printf("❌ Database resolution failed: %v\n", dbResult.Error())
	}

	loggerResult := di.ResolveProvider[LoggerService](container, context.Background(), "logger")
	if loggerResult.IsSuccess() {
		logger := loggerResult.Value()
		logger.Log("Test message")
		fmt.Printf("✅ Logger service working\n")
	} else {
		fmt.Printf("❌ Logger resolution failed: %v\n", loggerResult.Error())
	}

	// Test 2: Auto-wiring
	fmt.Println("\n🔧 Test 2: Auto-wiring Services")
	
	userService := &UserService{}
	err := container.Wire(userService)
	if err != nil {
		fmt.Printf("❌ Auto-wiring failed: %v\n", err)
	} else {
		if userService.DB != nil && userService.Logger != nil {
			result := userService.CreateUser("TestUser")
			fmt.Printf("✅ Auto-wired service result: %s\n", result)
		} else {
			fmt.Printf("❌ Services not wired properly\n")
		}
	}

	// Test 3: Interceptors
	fmt.Println("\n🎯 Test 3: Interceptors")
	
	container.AddInterceptor(func(ctx context.Context, serviceName string, next func() di.Result[any]) di.Result[any] {
		fmt.Printf("🔍 Intercepting resolution of: %s\n", serviceName)
		result := next()
		if result.IsSuccess() {
			fmt.Printf("✅ Successfully resolved: %s\n", serviceName)
		}
		return result
	})

	// Resolve with interceptor
	interceptedResult := di.ResolveProvider[DatabaseService](container, context.Background(), "database")
	if interceptedResult.IsSuccess() {
		fmt.Printf("✅ Interceptor working\n")
	}

	// Test 4: Container stats
	fmt.Println("\n📊 Test 4: Container Statistics")
	stats := container.GetStats()
	fmt.Printf("✅ Container stats: %+v\n", stats)

	fmt.Println("\n🎯 DI REAL TESTS COMPLETED")
}