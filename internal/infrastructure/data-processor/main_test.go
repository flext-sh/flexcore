// Package main provides comprehensive testing for Data Processor Plugin
package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/flext/flexcore/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataProcessor_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "empty config",
			config:  map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "config with mode",
			config: map[string]interface{}{
				"mode": "transform",
			},
			wantErr: false,
		},
		{
			name: "config with processing options",
			config: map[string]interface{}{
				"mode":          "filter",
				"batch_size":    100,
				"timeout":       30,
				"enable_cache":  true,
				"output_format": "json",
			},
			wantErr: false,
		},
		{
			name: "config with invalid mode type",
			config: map[string]interface{}{
				"mode": 123, // Should be string but plugin is tolerant
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DataProcessor{}
			ctx := context.Background()

			err := dp.Initialize(ctx, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.config, dp.GetConfig())
			assert.NotNil(t, dp.GetStatistics())
			assert.False(t, dp.GetStatistics().StartTime.IsZero())
			assert.Equal(t, int64(0), dp.GetStatistics().TotalRecords)
			assert.Equal(t, int64(0), dp.GetStatistics().ProcessedOK)
			assert.Equal(t, int64(0), dp.GetStatistics().ProcessedError)
		})
	}
}

func TestDataProcessor_Execute(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		input          map[string]interface{}
		wantErr        bool
		wantProcessor  string
		checkTimestamp bool
		checkData      bool
	}{
		{
			name:   "transform mode with array data",
			config: map[string]interface{}{"mode": "transform"},
			input: map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{"id": 1, "name": "item1"},
					map[string]interface{}{"id": 2, "name": "item2"},
				},
			},
			wantErr:        false,
			wantProcessor:  "data-processor",
			checkTimestamp: true,
			checkData:      true,
		},
		{
			name:   "filter mode with single record",
			config: map[string]interface{}{"mode": "filter"},
			input: map[string]interface{}{
				"data": map[string]interface{}{
					"id":   123,
					"name": "test_record",
					"type": "important",
				},
			},
			wantErr:        false,
			wantProcessor:  "data-processor",
			checkTimestamp: true,
			checkData:      true,
		},
		{
			name:   "enrich mode with string data",
			config: map[string]interface{}{"mode": "enrich"},
			input: map[string]interface{}{
				"data": "simple string data",
			},
			wantErr:        false,
			wantProcessor:  "data-processor",
			checkTimestamp: true,
			checkData:      true,
		},
		{
			name:   "validate mode with numeric data",
			config: map[string]interface{}{"mode": "validate"},
			input: map[string]interface{}{
				"data": 42.5,
			},
			wantErr:        false,
			wantProcessor:  "data-processor",
			checkTimestamp: true,
			checkData:      true,
		},
		{
			name:           "no mode specified with data",
			config:         map[string]interface{}{},
			input:          map[string]interface{}{"data": "test"},
			wantErr:        false,
			wantProcessor:  "data-processor",
			checkTimestamp: true,
			checkData:      true,
		},
		{
			name:           "no data field",
			config:         map[string]interface{}{"mode": "transform"},
			input:          map[string]interface{}{"other_field": "value"},
			wantErr:        false,
			wantProcessor:  "data-processor",
			checkTimestamp: true,
			checkData:      false,
		},
		{
			name:           "empty input",
			config:         map[string]interface{}{"mode": "transform"},
			input:          map[string]interface{}{},
			wantErr:        false,
			wantProcessor:  "data-processor",
			checkTimestamp: true,
			checkData:      false,
		},
		{
			name:           "nil input",
			config:         map[string]interface{}{"mode": "transform"},
			input:          nil,
			wantErr:        false,
			wantProcessor:  "data-processor",
			checkTimestamp: true,
			checkData:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DataProcessor{}
			ctx := context.Background()

			// Initialize first
			err := dp.Initialize(ctx, tt.config)
			require.NoError(t, err)

			result, err := dp.Execute(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Check processor field
			assert.Equal(t, tt.wantProcessor, result["processor"])

			// Check processed_by field
			assert.Contains(t, result["processed_by"], "FlexCore DataProcessor")

			// Check timestamp
			if tt.checkTimestamp {
				assert.NotNil(t, result["timestamp"])
				timestamp, ok := result["timestamp"].(int64)
				assert.True(t, ok)
				assert.Greater(t, timestamp, int64(0))
			}

			// Check data processing
			if tt.checkData {
				assert.NotNil(t, result["data"])
			}

			// Check stats
			assert.NotNil(t, result["stats"])
			statsMap, ok := result["stats"].(map[string]interface{})
			assert.True(t, ok)
			assert.NotNil(t, statsMap["total_records"])
			assert.NotNil(t, statsMap["processed_ok"])
			assert.NotNil(t, statsMap["processed_error"])
			assert.NotNil(t, statsMap["duration_ms"])
			assert.NotNil(t, statsMap["records_per_sec"])

			// Verify stats were updated
			assert.Equal(t, int64(1), dp.GetStatistics().ProcessedOK)
			assert.Equal(t, int64(1), dp.GetStatistics().TotalRecords)
			assert.Equal(t, int64(0), dp.GetStatistics().ProcessedError)
			assert.Greater(t, dp.GetStatistics().DurationMs, int64(0))
			assert.False(t, dp.GetStatistics().EndTime.IsZero())
		})
	}
}

func TestDataProcessor_ProcessingModes(t *testing.T) {
	modes := []string{"transform", "filter", "enrich", "validate", "aggregate"}

	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			dp := &DataProcessor{}
			ctx := context.Background()

			config := map[string]interface{}{"mode": mode}
			err := dp.Initialize(ctx, config)
			require.NoError(t, err)

			input := map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1,
					"name": "test_item",
					"mode": mode,
				},
			}

			result, err := dp.Execute(ctx, input)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "data-processor", result["processor"])
			assert.NotNil(t, result["data"])
		})
	}
}

func TestDataProcessor_DataTypes(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		wantType string
	}{
		{
			name: "array data",
			data: []interface{}{
				map[string]interface{}{"id": 1},
				map[string]interface{}{"id": 2},
			},
			wantType: "array",
		},
		{
			name: "object data",
			data: map[string]interface{}{
				"id":   123,
				"name": "test",
			},
			wantType: "object",
		},
		{
			name:     "string data",
			data:     "test string",
			wantType: "string",
		},
		{
			name:     "number data",
			data:     42.5,
			wantType: "number",
		},
		{
			name:     "boolean data",
			data:     true,
			wantType: "boolean",
		},
		{
			name:     "null data",
			data:     nil,
			wantType: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := &DataProcessor{}
			ctx := context.Background()

			err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
			require.NoError(t, err)

			input := map[string]interface{}{"data": tt.data}
			result, err := dp.Execute(ctx, input)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "data-processor", result["processor"])

			// Verify data was processed
			if tt.data != nil {
				assert.NotNil(t, result["data"])
			}
		})
	}
}

func TestDataProcessor_GetInfo(t *testing.T) {
	t.Run("plugin info is correct", func(t *testing.T) {
		dp := &DataProcessor{}
		info := dp.GetInfo()

		assert.Equal(t, "data-processor", info.ID)
		assert.Equal(t, "data-processor", info.Name)
		assert.Equal(t, "0.9.0", info.Version)
		assert.Contains(t, info.Description, "Data processing plugin")
		assert.Equal(t, "FlexCore Team", info.Author)
		assert.Equal(t, "processor", info.Type)
		assert.Contains(t, info.Tags, "data")
		assert.Contains(t, info.Tags, "processor")
		assert.Equal(t, "active", info.Status)
		assert.Equal(t, "healthy", info.Health)
		assert.False(t, info.LoadedAt.IsZero())
	})
}

func TestDataProcessor_HealthCheck(t *testing.T) {
	t.Run("health check succeeds", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()

		// Initialize and process some data
		err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
		require.NoError(t, err)

		_, err = dp.Execute(ctx, map[string]interface{}{"data": "test"})
		require.NoError(t, err)

		// Health check should succeed
		err = dp.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("health check with zero records", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()

		// Initialize but don't process any data
		err := dp.Initialize(ctx, map[string]interface{}{"mode": "filter"})
		require.NoError(t, err)

		// Health check should still succeed
		err = dp.HealthCheck(ctx)
		assert.NoError(t, err)
	})
}

func TestDataProcessor_Cleanup(t *testing.T) {
	t.Run("cleanup without stats file", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()

		// Initialize without stats file
		err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
		require.NoError(t, err)

		// Process some data
		_, err = dp.Execute(ctx, map[string]interface{}{"data": "test"})
		require.NoError(t, err)

		// Cleanup should succeed
		err = dp.Cleanup()
		assert.NoError(t, err)
	})

	t.Run("cleanup with stats file", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()
		statsFile := "/tmp/test_data_processor_stats.json"

		// Clean up any existing file
		os.Remove(statsFile)
		defer os.Remove(statsFile)

		// Initialize with stats file
		err := dp.Initialize(ctx, map[string]interface{}{
			"mode":       "transform",
			"stats_file": statsFile,
		})
		require.NoError(t, err)

		// Process some data
		_, err = dp.Execute(ctx, map[string]interface{}{"data": "test"})
		require.NoError(t, err)

		// Cleanup should succeed and create stats file
		err = dp.Cleanup()
		assert.NoError(t, err)

		// Check that stats file was created
		_, err = os.Stat(statsFile)
		assert.NoError(t, err)

		// Verify stats file content
		data, err := os.ReadFile(statsFile)
		require.NoError(t, err)

		var stats plugin.ProcessingStats
		err = json.Unmarshal(data, &stats)
		require.NoError(t, err)

		assert.Equal(t, int64(1), stats.TotalRecords)
		assert.Equal(t, int64(1), stats.ProcessedOK)
		assert.Equal(t, int64(0), stats.ProcessedError)
	})
}

func TestDataProcessor_StatsTracking(t *testing.T) {
	t.Run("stats are tracked correctly", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()

		// Initialize
		err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
		require.NoError(t, err)

		// Verify initial stats
		assert.Equal(t, int64(0), dp.GetStatistics().TotalRecords)
		assert.Equal(t, int64(0), dp.GetStatistics().ProcessedOK)
		assert.Equal(t, int64(0), dp.GetStatistics().ProcessedError)
		assert.False(t, dp.GetStatistics().StartTime.IsZero())

		// Process multiple items
		for i := 0; i < 10; i++ {
			_, err = dp.Execute(ctx, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   i,
					"type": "test_record",
				},
			})
			require.NoError(t, err)
		}

		// Verify final stats
		assert.Equal(t, int64(10), dp.GetStatistics().TotalRecords)
		assert.Equal(t, int64(10), dp.GetStatistics().ProcessedOK)
		assert.Equal(t, int64(0), dp.GetStatistics().ProcessedError)
		assert.False(t, dp.GetStatistics().EndTime.IsZero())
		assert.Greater(t, dp.GetStatistics().DurationMs, int64(0))
	})
}

func TestDataProcessor_ConfigurationHandling(t *testing.T) {
	t.Run("complex configuration", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()

		complexConfig := map[string]interface{}{
			"mode":          "transform",
			"batch_size":    500,
			"timeout":       60,
			"enable_cache":  true,
			"output_format": "json",
			"filters": map[string]interface{}{
				"include_types": []string{"important", "critical"},
				"exclude_ids":   []int{1, 2, 3},
			},
			"transformations": []map[string]interface{}{
				{"field": "name", "operation": "uppercase"},
				{"field": "timestamp", "operation": "format"},
			},
		}

		err := dp.Initialize(ctx, complexConfig)
		require.NoError(t, err)
		assert.Equal(t, complexConfig, dp.GetConfig())

		// Verify processor can handle complex data
		input := map[string]interface{}{
			"data": map[string]interface{}{
				"id":        123,
				"name":      "test_item",
				"type":      "important",
				"timestamp": time.Now().Unix(),
			},
		}

		result, err := dp.Execute(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "data-processor", result["processor"])
	})
}

func TestDataProcessor_EdgeCases(t *testing.T) {
	t.Run("very large dataset", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()

		err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
		require.NoError(t, err)

		// Create large dataset
		largeDataset := make([]interface{}, 5000)
		for i := range largeDataset {
			largeDataset[i] = map[string]interface{}{
				"id":        i,
				"data":      strings.Repeat("x", 100), // 100 chars per record
				"timestamp": time.Now().Unix(),
			}
		}

		result, err := dp.Execute(ctx, map[string]interface{}{
			"data": largeDataset,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "data-processor", result["processor"])
	})

	t.Run("deeply nested data structures", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()

		err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
		require.NoError(t, err)

		deeplyNested := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": map[string]interface{}{
						"level4": map[string]interface{}{
							"level5": map[string]interface{}{
								"value": "deep_nested_value",
								"array": []interface{}{1, 2, 3, 4, 5},
							},
						},
					},
				},
			},
		}

		result, err := dp.Execute(ctx, map[string]interface{}{
			"data": deeplyNested,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("unicode and special characters", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()

		err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
		require.NoError(t, err)

		unicodeData := map[string]interface{}{
			"chinese":       "中文数据处理",
			"japanese":      "データ処理システム",
			"arabic":        "معالجة البيانات",
			"emoji":         "🚀📊🔧🎉",
			"special_chars": "!@#$%^&*(){}[]|\\:;\"'<>?,./-=+~`",
			"newlines":      "line1\nline2\r\nline3\ttabbed",
		}

		result, err := dp.Execute(ctx, map[string]interface{}{
			"data": unicodeData,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify unicode data is preserved
		processedData := result["data"].(map[string]interface{})
		assert.Equal(t, "中文数据处理", processedData["chinese"])
		assert.Equal(t, "データ処理システム", processedData["japanese"])
		assert.Equal(t, "معالجة البيانات", processedData["arabic"])
		assert.Equal(t, "🚀📊🔧🎉", processedData["emoji"])
	})
}

// Test concurrent processing to ensure thread safety
func TestDataProcessor_Concurrent(t *testing.T) {
	t.Run("concurrent processing", func(t *testing.T) {
		dp := &DataProcessor{}
		ctx := context.Background()

		err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
		require.NoError(t, err)

		const numGoroutines = 100
		const numOperations = 5

		results := make(chan error, numGoroutines*numOperations)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					_, err := dp.Execute(ctx, map[string]interface{}{
						"data": map[string]interface{}{
							"goroutine_id": id,
							"operation_id": j,
							"timestamp":    time.Now().Unix(),
							"mode":         "concurrent_test",
						},
					})
					results <- err
				}
			}(i)
		}

		// Collect all results
		for i := 0; i < numGoroutines*numOperations; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent processing failed")
		}

		// Verify final stats
		expectedTotal := int64(numGoroutines * numOperations)
		assert.Equal(t, expectedTotal, dp.GetStatistics().TotalRecords)
		assert.Equal(t, expectedTotal, dp.GetStatistics().ProcessedOK)
		assert.Equal(t, int64(0), dp.GetStatistics().ProcessedError)
	})
}

// Benchmark tests for performance validation
func BenchmarkDataProcessor_Execute(b *testing.B) {
	dp := &DataProcessor{}
	ctx := context.Background()

	err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
	if err != nil {
		b.Fatal(err)
	}

	testData := map[string]interface{}{
		"data": map[string]interface{}{
			"id":        12345,
			"name":      "benchmark_item",
			"type":      "performance_test",
			"timestamp": time.Now().Unix(),
			"metadata": map[string]interface{}{
				"source":  "benchmark",
				"version": "1.0",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dp.Execute(ctx, testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataProcessor_LargeArray(b *testing.B) {
	dp := &DataProcessor{}
	ctx := context.Background()

	err := dp.Initialize(ctx, map[string]interface{}{"mode": "transform"})
	if err != nil {
		b.Fatal(err)
	}

	// Create test array
	testArray := make([]interface{}, 1000)
	for i := range testArray {
		testArray[i] = map[string]interface{}{
			"id":   i,
			"name": "item",
			"type": "benchmark",
		}
	}

	testData := map[string]interface{}{"data": testArray}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dp.Execute(ctx, testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataProcessor_Initialize(b *testing.B) {
	config := map[string]interface{}{
		"mode":         "transform",
		"batch_size":   1000,
		"enable_cache": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dp := &DataProcessor{}
		_ = dp.Initialize(context.Background(), config)
	}
}
