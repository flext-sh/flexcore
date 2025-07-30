// Package adapter provides base functionality for all FlexCore adapters
package adapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flext/flexcore/pkg/result"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseAdapter(t *testing.T) {
	tests := []struct {
		name        string
		adapterName string
		version     string
		opts        []AdapterOption
		wantName    string
		wantVersion string
	}{
		{
			name:        "basic adapter creation",
			adapterName: "test-adapter",
			version:     "0.9.0",
			wantName:    "test-adapter",
			wantVersion: "0.9.0",
		},
		{
			name:        "adapter with hooks",
			adapterName: "test-adapter-hooks",
			version:     "2.0.0",
			opts: []AdapterOption{
				WithHooks(AdapterHooks{
					OnError: func(ctx context.Context, err error) {
						// Test hook
					},
				}),
			},
			wantName:    "test-adapter-hooks",
			wantVersion: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBaseAdapter(tt.adapterName, tt.version, tt.opts...)
			require.NotNil(t, adapter)
			assert.Equal(t, tt.wantName, adapter.Name())
			assert.Equal(t, tt.wantVersion, adapter.Version())
			assert.NotNil(t, adapter.validator)
			assert.NotNil(t, adapter.tracer)
		})
	}
}

func TestBaseAdapter_Configure(t *testing.T) {
	type testConfig struct {
		Host     string `validate:"required"`
		Port     int    `validate:"required,min=1,max=65535"`
		Database string `validate:"required"`
	}

	tests := []struct {
		name      string
		rawConfig map[string]interface{}
		wantErr   bool
		wantHost  string
		wantPort  int
	}{
		{
			name: "valid configuration",
			rawConfig: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"database": "testdb",
			},
			wantHost: "localhost",
			wantPort: 5432,
		},
		{
			name: "missing required field",
			rawConfig: map[string]interface{}{
				"host": "localhost",
				"port": 5432,
				// missing database
			},
			wantErr: true,
		},
		{
			name: "invalid port range",
			rawConfig: map[string]interface{}{
				"host":     "localhost",
				"port":     70000, // invalid port
				"database": "testdb",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBaseAdapter("test", "0.9.0")
			var config testConfig
			err := adapter.Configure(&config, tt.rawConfig)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, config.Host)
			assert.Equal(t, tt.wantPort, config.Port)
			assert.Equal(t, config, adapter.Config())
		})
	}
}

func TestBaseAdapter_ExtractWithHooks(t *testing.T) {
	ctx := context.Background()
	req := ExtractRequest{
		Source: "test-source",
		Limit:  100,
	}

	tests := []struct {
		name        string
		hooks       AdapterHooks
		extractFn   func(context.Context, ExtractRequest) result.Result[*ExtractResponse]
		wantErr     bool
		wantRecords int
		hooksCalled map[string]bool
	}{
		{
			name: "successful extraction with hooks",
			hooks: AdapterHooks{
				OnBeforeExtract: func(ctx context.Context, req ExtractRequest) error {
					return nil
				},
				OnAfterExtract: func(ctx context.Context, req ExtractRequest, resp *ExtractResponse, err error) {
					// Hook called
				},
			},
			extractFn: func(ctx context.Context, req ExtractRequest) result.Result[*ExtractResponse] {
				return result.Success(&ExtractResponse{
					Records: []Record{
						{ID: "1", Data: map[string]interface{}{"test": "data"}, Timestamp: time.Now()},
						{ID: "2", Data: map[string]interface{}{"test": "data2"}, Timestamp: time.Now()},
					},
					ExtractedAt: time.Now(),
				})
			},
			wantRecords: 2,
		},
		{
			name: "extraction failure with error hook",
			hooks: AdapterHooks{
				OnError: func(ctx context.Context, err error) {
					// Error hook called
				},
			},
			extractFn: func(ctx context.Context, req ExtractRequest) result.Result[*ExtractResponse] {
				return result.Failure[*ExtractResponse](errors.New("extraction failed"))
			},
			wantErr: true,
		},
		{
			name: "before hook failure",
			hooks: AdapterHooks{
				OnBeforeExtract: func(ctx context.Context, req ExtractRequest) error {
					return errors.New("before hook failed")
				},
			},
			extractFn: func(ctx context.Context, req ExtractRequest) result.Result[*ExtractResponse] {
				// Should not be called
				return result.Success(&ExtractResponse{})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBaseAdapter("test", "0.9.0", WithHooks(tt.hooks))

			resp, err := adapter.ExtractWithHooks(ctx, req, tt.extractFn)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Len(t, resp.Records, tt.wantRecords)
		})
	}
}

func TestBaseAdapter_LoadWithHooks(t *testing.T) {
	ctx := context.Background()
	req := LoadRequest{
		Target: "test-target",
		Records: []Record{
			{ID: "1", Data: map[string]interface{}{"test": "data"}, Timestamp: time.Now()},
		},
		Mode: LoadModeAppend,
	}

	tests := []struct {
		name       string
		hooks      AdapterHooks
		loadFn     func(context.Context, LoadRequest) result.Result[*LoadResponse]
		wantErr    bool
		wantLoaded int64
		wantFailed int64
	}{
		{
			name: "successful load with hooks",
			hooks: AdapterHooks{
				OnBeforeLoad: func(ctx context.Context, req LoadRequest) error {
					return nil
				},
				OnAfterLoad: func(ctx context.Context, req LoadRequest, resp *LoadResponse, err error) {
					// Hook called
				},
			},
			loadFn: func(ctx context.Context, req LoadRequest) result.Result[*LoadResponse] {
				return result.Success(&LoadResponse{
					RecordsLoaded: 1,
					RecordsFailed: 0,
					LoadedAt:      time.Now(),
				})
			},
			wantLoaded: 1,
			wantFailed: 0,
		},
		{
			name: "load failure with error hook",
			hooks: AdapterHooks{
				OnError: func(ctx context.Context, err error) {
					// Error hook called
				},
			},
			loadFn: func(ctx context.Context, req LoadRequest) result.Result[*LoadResponse] {
				return result.Failure[*LoadResponse](errors.New("load failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewBaseAdapter("test", "0.9.0", WithHooks(tt.hooks))

			resp, err := adapter.LoadWithHooks(ctx, req, tt.loadFn)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantLoaded, resp.RecordsLoaded)
			assert.Equal(t, tt.wantFailed, resp.RecordsFailed)
		})
	}
}

func TestAdapterOptions(t *testing.T) {
	t.Run("WithHooks option", func(t *testing.T) {
		hooks := AdapterHooks{
			OnError: func(ctx context.Context, err error) {},
		}

		adapter := NewBaseAdapter("test", "0.9.0", WithHooks(hooks))
		assert.NotNil(t, adapter.hooks.OnError)
	})

	t.Run("WithValidator option", func(t *testing.T) {
		customValidator := &validator.Validate{}
		adapter := NewBaseAdapter("test", "0.9.0", WithValidator(customValidator))
		assert.Equal(t, customValidator, adapter.validator)
	})
}

// Benchmark tests for performance validation
func BenchmarkBaseAdapter_ExtractWithHooks(b *testing.B) {
	ctx := context.Background()
	req := ExtractRequest{
		Source: "benchmark-source",
		Limit:  1000,
	}

	adapter := NewBaseAdapter("benchmark", "0.9.0")
	extractFn := func(ctx context.Context, req ExtractRequest) result.Result[*ExtractResponse] {
		records := make([]Record, req.Limit)
		for i := 0; i < req.Limit; i++ {
			records[i] = Record{
				ID:        string(rune(i)),
				Data:      map[string]interface{}{"index": i},
				Timestamp: time.Now(),
			}
		}
		return result.Success(&ExtractResponse{
			Records:     records,
			ExtractedAt: time.Now(),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := adapter.ExtractWithHooks(ctx, req, extractFn)
		if err != nil {
			b.Fatal(err)
		}
	}
}
