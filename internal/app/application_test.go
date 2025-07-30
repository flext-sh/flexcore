// Package app contains comprehensive tests for the application layer
package app

import (
	"context"
	"testing"
	"time"

	"github.com/flext/flexcore/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApplication(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: &config.Config{
				App: struct {
					Name        string `mapstructure:"name"`
					Version     string `mapstructure:"version"`
					Environment string `mapstructure:"environment"`
					Debug       bool   `mapstructure:"debug"`
					Port        int    `mapstructure:"port"`
				}{
					Name:        "flexcore",
					Version:     "0.9.0",
					Environment: "test",
					Debug:       true,
					Port:        8080,
				},
				Server: struct {
					Host         string        `mapstructure:"host"`
					Port         int           `mapstructure:"port"`
					ReadTimeout  time.Duration `mapstructure:"read_timeout"`
					WriteTimeout time.Duration `mapstructure:"write_timeout"`
					IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
				}{
					Host:         "localhost",
					Port:         8080,
					ReadTimeout:  15 * time.Second,
					WriteTimeout: 15 * time.Second,
					IdleTimeout:  60 * time.Second,
				},
			},
			wantError: false,
		},
		{
			name: "missing environment",
			config: &config.Config{
				App: struct {
					Name        string `mapstructure:"name"`
					Version     string `mapstructure:"version"`
					Environment string `mapstructure:"environment"`
					Debug       bool   `mapstructure:"debug"`
					Port        int    `mapstructure:"port"`
				}{
					Name:        "flexcore",
					Version:     "0.9.0",
					Environment: "", // Missing environment
					Debug:       true,
					Port:        8080,
				},
				Server: struct {
					Host         string        `mapstructure:"host"`
					Port         int           `mapstructure:"port"`
					ReadTimeout  time.Duration `mapstructure:"read_timeout"`
					WriteTimeout time.Duration `mapstructure:"write_timeout"`
					IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
				}{
					Host:         "localhost",
					Port:         8080,
					ReadTimeout:  15 * time.Second,
					WriteTimeout: 15 * time.Second,
					IdleTimeout:  60 * time.Second,
				},
			},
			wantError: true,
			errorMsg:  "environment is required",
		},
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
			errorMsg:  "config cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApplication(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, app)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, app)
				assert.Equal(t, tt.config, app.Config())
			}
		})
	}
}

func TestApplication_StartStop(t *testing.T) {
	cfg := &config.Config{
		App: struct {
			Name        string `mapstructure:"name"`
			Version     string `mapstructure:"version"`
			Environment string `mapstructure:"environment"`
			Debug       bool   `mapstructure:"debug"`
			Port        int    `mapstructure:"port"`
		}{
			Name:        "flexcore",
			Version:     "0.9.0",
			Environment: "test",
			Debug:       true,
			Port:        8080,
		},
		Server: struct {
			Host         string        `mapstructure:"host"`
			Port         int           `mapstructure:"port"`
			ReadTimeout  time.Duration `mapstructure:"read_timeout"`
			WriteTimeout time.Duration `mapstructure:"write_timeout"`
			IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
		}{
			Host:         "localhost",
			Port:         8081, // Different port to avoid conflicts
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	app, err := NewApplication(cfg)
	require.NoError(t, err)
	require.NotNil(t, app)

	ctx := context.Background()

	// Test start
	err = app.Start(ctx)
	assert.NoError(t, err)

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test stop
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = app.Stop(stopCtx)
	assert.NoError(t, err)
}
