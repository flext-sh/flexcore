package flexcore

import (
	"testing"
	"time"

	"github.com/flext/flexcore/infrastructure/di"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKernelCreation(t *testing.T) {
	kernel := NewKernel(
		WithAppName("test-app"),
		WithVersion("1.0.0"),
		WithDebug(),
		WithEventsURL("http://localhost:8000"),
	)

	assert.NotNil(t, kernel)
	assert.Equal(t, "test-app", kernel.config.AppName)
	assert.Equal(t, "1.0.0", kernel.config.Version)
	assert.True(t, kernel.config.Debug)
	assert.Equal(t, "http://localhost:8000", kernel.config.EventsURL)
}

func TestApplicationBuild(t *testing.T) {
	kernel := NewKernel(WithAppName("test-app"))
	container := di.NewContainer()
	
	app := kernel.BuildApplication(container)
	
	assert.NotNil(t, app)
	assert.Equal(t, kernel, app.kernel)
	assert.Equal(t, container, app.container)
}

func TestApplicationInitialization(t *testing.T) {
	kernel := NewKernel(
		WithAppName("test-app"),
		WithEventsURL("http://localhost:8000"),
	)
	container := di.NewContainer()
	
	app := kernel.BuildApplication(container)
	
	// Test infrastructure initialization
	err := app.initializeInfrastructure()
	assert.NoError(t, err)
	
	// Test domain initialization
	err = app.initializeDomain()
	assert.NoError(t, err)
	
	// Test application initialization
	err = app.initializeApplication()
	assert.NoError(t, err)
}

func TestApplicationLifecycle(t *testing.T) {
	kernel := NewKernel(
		WithAppName("test-app"),
		WithEventsURL("http://localhost:8000"),
	)
	container := di.NewContainer()
	
	app := kernel.BuildApplication(container)
	
	// Start application in goroutine (it blocks on ctx.Done())
	go func() {
		err := app.Run()
		// Error is expected when we stop the app
		assert.NoError(t, err)
	}()
	
	// Give it time to start
	time.Sleep(100 * time.Millisecond)
	
	// Stop application
	app.Stop()
	
	// Give it time to stop
	time.Sleep(100 * time.Millisecond)
}

func TestContainerIntegration(t *testing.T) {
	kernel := NewKernel(WithEventsURL("http://localhost:8000"))
	container := di.NewContainer()
	
	app := kernel.BuildApplication(container)
	
	// Initialize all layers
	err := app.initializeInfrastructure()
	require.NoError(t, err)
	
	err = app.initializeDomain()
	require.NoError(t, err)
	
	err = app.initializeApplication()
	require.NoError(t, err)
	
	// Verify services are registered
	appContainer := app.Container()
	assert.NotNil(t, appContainer)
	
	// Check that services exist in container
	services := appContainer.GetRegisteredServices()
	assert.Greater(t, len(services), 0)
}