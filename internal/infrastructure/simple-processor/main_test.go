// Package main provides comprehensive testing for Simple Processor Plugin
package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleProcessor_Initialize(t *testing.T) {
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
			name: "config with stats file",
			config: map[string]interface{}{
				"stats_file": "/tmp/test_stats.json",
			},
			wantErr: false,
		},
		{
			name: "config with various settings",
			config: map[string]interface{}{
				"batch_size": 100,
				"timeout":    30,
				"debug":      true,
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
			sp := &SimpleProcessor{}
			ctx := context.Background()

			err := sp.Initialize(ctx, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.config, sp.config)
			assert.False(t, sp.stats.StartTime.IsZero())
			assert.Equal(t, int64(0), sp.stats.TotalRecords)
			assert.Equal(t, int64(0), sp.stats.ProcessedOK)
			assert.Equal(t, int64(0), sp.stats.ProcessedError)
		})
	}
}

func TestSimpleProcessor_Execute(t *testing.T) {
	tests := []struct {
		name           string
		input          map[string]interface{}
		wantErr        bool
		wantProcessor  string
		wantRecords    int
		checkTimestamp bool
	}{
		{
			name: "process array data",
			input: map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{"id": 1, "name": "item1"},
					map[string]interface{}{"id": 2, "name": "item2"},
					map[string]interface{}{"id": 3, "name": "item3"},
				},
			},
			wantErr:        false,
			wantProcessor:  "simple-processor",
			wantRecords:    3,
			checkTimestamp: true,
		},
		{
			name: "process map data",
			input: map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1,
					"name": "test_item",
					"type": "test",
				},
			},
			wantErr:        false,
			wantProcessor:  "simple-processor",
			wantRecords:    1,
			checkTimestamp: true,
		},
		{
			name: "process string data",
			input: map[string]interface{}{
				"data": "test string data",
			},
			wantErr:        false,
			wantProcessor:  "simple-processor",
			wantRecords:    1,
			checkTimestamp: true,
		},
		{
			name: "process other data type",
			input: map[string]interface{}{
				"data": 12345,
			},
			wantErr:        false,
			wantProcessor:  "simple-processor",
			wantRecords:    1,
			checkTimestamp: true,
		},
		{
			name:           "no data field",
			input:          map[string]interface{}{"other_field": "value"},
			wantErr:        false,
			wantProcessor:  "simple-processor",
			wantRecords:    0, // No records_count when no data
			checkTimestamp: true,
		},
		{
			name:           "empty input",
			input:          map[string]interface{}{},
			wantErr:        false,
			wantProcessor:  "simple-processor",
			wantRecords:    0,
			checkTimestamp: true,
		},
		{
			name:           "nil input",
			input:          nil,
			wantErr:        false,
			wantProcessor:  "simple-processor",
			wantRecords:    0,
			checkTimestamp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp := &SimpleProcessor{}
			ctx := context.Background()

			// Initialize first
			err := sp.Initialize(ctx, map[string]interface{}{})
			require.NoError(t, err)

			result, err := sp.Execute(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Check processor field
			assert.Equal(t, tt.wantProcessor, result["processor"])

			// Check processed_by field
			assert.Equal(t, "FlexCore SimpleProcessor v1.0", result["processed_by"])

			// Check timestamp
			if tt.checkTimestamp {
				assert.NotNil(t, result["timestamp"])
				timestamp, ok := result["timestamp"].(int64)
				assert.True(t, ok)
				assert.Greater(t, timestamp, int64(0))
			}

			// Check records count if applicable
			if tt.wantRecords > 0 {
				assert.Equal(t, tt.wantRecords, result["records_count"])
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
			assert.Equal(t, int64(1), sp.stats.ProcessedOK)
			assert.Equal(t, int64(1), sp.stats.TotalRecords)
			assert.Equal(t, int64(0), sp.stats.ProcessedError)
			assert.Greater(t, sp.stats.DurationMs, int64(0))
			assert.False(t, sp.stats.EndTime.IsZero())
		})
	}
}

func TestSimpleProcessor_ProcessArray(t *testing.T) {
	t.Run("process array with map items", func(t *testing.T) {
		sp := &SimpleProcessor{}
		input := []interface{}{
			map[string]interface{}{"id": 1, "name": "item1"},
			map[string]interface{}{"id": 2, "name": "item2"},
			"string_item", // Non-map item
			42,            // Non-map item
		}

		result := sp.processArray(input)
		require.Len(t, result, 4)

		// Check first map item (should have metadata added)
		firstItem, ok := result[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 1, firstItem["id"])
		assert.Equal(t, "item1", firstItem["name"])
		assert.NotNil(t, firstItem["_processed_at"])
		assert.Equal(t, "simple-processor", firstItem["_processor"])

		// Check second map item
		secondItem, ok := result[1].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 2, secondItem["id"])
		assert.Equal(t, "item2", secondItem["name"])
		assert.NotNil(t, secondItem["_processed_at"])
		assert.Equal(t, "simple-processor", secondItem["_processor"])

		// Check non-map items (should remain unchanged)
		assert.Equal(t, "string_item", result[2])
		assert.Equal(t, 42, result[3])
	})

	t.Run("process empty array", func(t *testing.T) {
		sp := &SimpleProcessor{}
		input := []interface{}{}

		result := sp.processArray(input)
		assert.Empty(t, result)
		assert.Len(t, result, 0)
	})
}

func TestSimpleProcessor_ProcessMap(t *testing.T) {
	t.Run("process map data", func(t *testing.T) {
		sp := &SimpleProcessor{}
		input := map[string]interface{}{
			"id":   123,
			"name": "test_item",
			"type": "test",
			"tags": []string{"tag1", "tag2"},
		}

		result := sp.processMap(input)
		require.NotNil(t, result)

		// Check original data is preserved
		assert.Equal(t, 123, result["id"])
		assert.Equal(t, "test_item", result["name"])
		assert.Equal(t, "test", result["type"])
		assert.Equal(t, []string{"tag1", "tag2"}, result["tags"])

		// Check metadata is added
		assert.NotNil(t, result["_processed_at"])
		assert.Equal(t, "simple-processor", result["_processor"])

		// Check timestamp is valid
		processedAt, ok := result["_processed_at"].(int64)
		assert.True(t, ok)
		assert.Greater(t, processedAt, int64(0))
	})

	t.Run("process empty map", func(t *testing.T) {
		sp := &SimpleProcessor{}
		input := map[string]interface{}{}

		result := sp.processMap(input)
		require.NotNil(t, result)

		// Should only have metadata
		assert.Equal(t, "simple-processor", result["_processor"])
		assert.NotNil(t, result["_processed_at"])
		assert.Len(t, result, 2) // Only metadata fields
	})
}

func TestSimpleProcessor_ProcessString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "test string",
			expected: "[PROCESSED] test string",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "[PROCESSED] ",
		},
		{
			name:     "string with special characters",
			input:    "special chars: !@#$%^&*()",
			expected: "[PROCESSED] special chars: !@#$%^&*()",
		},
		{
			name:     "unicode string",
			input:    "unicode: 测试数据 🚀",
			expected: "[PROCESSED] unicode: 测试数据 🚀",
		},
		{
			name:     "multiline string",
			input:    "line1\nline2\nline3",
			expected: "[PROCESSED] line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp := &SimpleProcessor{}
			result := sp.processString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleProcessor_GetInfo(t *testing.T) {
	t.Run("plugin info is correct", func(t *testing.T) {
		sp := &SimpleProcessor{}
		info := sp.GetInfo()

		assert.Equal(t, "simple-processor", info.ID)
		assert.Equal(t, "simple-processor", info.Name)
		assert.Equal(t, "1.0.0", info.Version)
		assert.Equal(t, "Simple data processing plugin for FlexCore testing", info.Description)
		assert.Equal(t, "FlexCore Team", info.Author)
		assert.Equal(t, "processor", info.Type)
		assert.Equal(t, []string{"filter", "transform", "enrich"}, info.Tags)
		assert.Equal(t, "active", info.Status)
		assert.Equal(t, "healthy", info.Health)
		assert.False(t, info.LoadedAt.IsZero())
	})
}

func TestSimpleProcessor_HealthCheck(t *testing.T) {
	t.Run("health check succeeds", func(t *testing.T) {
		sp := &SimpleProcessor{}
		ctx := context.Background()

		// Initialize and process some data
		err := sp.Initialize(ctx, map[string]interface{}{})
		require.NoError(t, err)

		_, err = sp.Execute(ctx, map[string]interface{}{"data": "test"})
		require.NoError(t, err)

		// Health check should succeed
		err = sp.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("health check with zero records", func(t *testing.T) {
		sp := &SimpleProcessor{}
		ctx := context.Background()

		// Initialize but don't process any data
		err := sp.Initialize(ctx, map[string]interface{}{})
		require.NoError(t, err)

		// Health check should still succeed
		err = sp.HealthCheck(ctx)
		assert.NoError(t, err)
	})
}

func TestSimpleProcessor_Cleanup(t *testing.T) {
	t.Run("cleanup without stats file", func(t *testing.T) {
		sp := &SimpleProcessor{}
		ctx := context.Background()

		// Initialize without stats file
		err := sp.Initialize(ctx, map[string]interface{}{})
		require.NoError(t, err)

		// Process some data
		_, err = sp.Execute(ctx, map[string]interface{}{"data": "test"})
		require.NoError(t, err)

		// Cleanup should succeed
		err = sp.Cleanup()
		assert.NoError(t, err)
	})

	t.Run("cleanup with stats file", func(t *testing.T) {
		sp := &SimpleProcessor{}
		ctx := context.Background()
		statsFile := "/tmp/test_simple_processor_stats.json"

		// Clean up any existing file
		os.Remove(statsFile)
		defer os.Remove(statsFile)

		// Initialize with stats file
		err := sp.Initialize(ctx, map[string]interface{}{
			"stats_file": statsFile,
		})
		require.NoError(t, err)

		// Process some data
		_, err = sp.Execute(ctx, map[string]interface{}{"data": "test"})
		require.NoError(t, err)

		// Cleanup should succeed and create stats file
		err = sp.Cleanup()
		assert.NoError(t, err)

		// Check that stats file was created
		_, err = os.Stat(statsFile)
		assert.NoError(t, err)

		// Verify stats file content
		data, err := os.ReadFile(statsFile)
		require.NoError(t, err)

		var stats ProcessingStats
		err = json.Unmarshal(data, &stats)
		require.NoError(t, err)

		assert.Equal(t, int64(1), stats.TotalRecords)
		assert.Equal(t, int64(1), stats.ProcessedOK)
		assert.Equal(t, int64(0), stats.ProcessedError)
	})
}

func TestSimpleProcessor_StatsTracking(t *testing.T) {
	t.Run("stats are tracked correctly", func(t *testing.T) {
		sp := &SimpleProcessor{}
		ctx := context.Background()

		// Initialize
		err := sp.Initialize(ctx, map[string]interface{}{})
		require.NoError(t, err)

		// Verify initial stats
		assert.Equal(t, int64(0), sp.stats.TotalRecords)
		assert.Equal(t, int64(0), sp.stats.ProcessedOK)
		assert.Equal(t, int64(0), sp.stats.ProcessedError)
		assert.False(t, sp.stats.StartTime.IsZero())

		// Process multiple items
		for i := 0; i < 5; i++ {
			_, err = sp.Execute(ctx, map[string]interface{}{
				"data": map[string]interface{}{"id": i},
			})
			require.NoError(t, err)
		}

		// Verify final stats
		assert.Equal(t, int64(5), sp.stats.TotalRecords)
		assert.Equal(t, int64(5), sp.stats.ProcessedOK)
		assert.Equal(t, int64(0), sp.stats.ProcessedError)
		assert.False(t, sp.stats.EndTime.IsZero())
		assert.Greater(t, sp.stats.DurationMs, int64(0))
	})
}

func TestSimpleProcessor_EdgeCases(t *testing.T) {
	t.Run("very large array", func(t *testing.T) {
		sp := &SimpleProcessor{}
		ctx := context.Background()

		err := sp.Initialize(ctx, map[string]interface{}{})
		require.NoError(t, err)

		// Create large array
		largeArray := make([]interface{}, 1000)
		for i := range largeArray {
			largeArray[i] = map[string]interface{}{"id": i, "data": "item"}
		}

		result, err := sp.Execute(ctx, map[string]interface{}{
			"data": largeArray,
		})
		require.NoError(t, err)
		assert.Equal(t, 1000, result["records_count"])
	})

	t.Run("deeply nested data", func(t *testing.T) {
		sp := &SimpleProcessor{}
		ctx := context.Background()

		err := sp.Initialize(ctx, map[string]interface{}{})
		require.NoError(t, err)

		nestedData := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": map[string]interface{}{
						"value": "deep_value",
					},
				},
			},
		}

		result, err := sp.Execute(ctx, map[string]interface{}{
			"data": nestedData,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, result["records_count"])
	})

	t.Run("unicode data processing", func(t *testing.T) {
		sp := &SimpleProcessor{}
		ctx := context.Background()

		err := sp.Initialize(ctx, map[string]interface{}{})
		require.NoError(t, err)

		unicodeData := map[string]interface{}{
			"chinese":  "中文数据",
			"japanese": "日本語データ",
			"arabic":   "بيانات عربية",
			"emoji":    "🚀🎉📊",
		}

		result, err := sp.Execute(ctx, map[string]interface{}{
			"data": unicodeData,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, result["records_count"])

		// Verify unicode data is preserved
		processedData := result["data"].(map[string]interface{})
		assert.Equal(t, "中文数据", processedData["chinese"])
		assert.Equal(t, "日本語データ", processedData["japanese"])
		assert.Equal(t, "بيانات عربية", processedData["arabic"])
		assert.Equal(t, "🚀🎉📊", processedData["emoji"])
	})
}

// Test concurrent processing to ensure thread safety
func TestSimpleProcessor_Concurrent(t *testing.T) {
	t.Run("concurrent processing", func(t *testing.T) {
		sp := &SimpleProcessor{}
		ctx := context.Background()

		err := sp.Initialize(ctx, map[string]interface{}{})
		require.NoError(t, err)

		const numGoroutines = 50
		const numOperations = 10

		results := make(chan error, numGoroutines*numOperations)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					_, err := sp.Execute(ctx, map[string]interface{}{
						"data": map[string]interface{}{
							"goroutine_id": id,
							"operation_id": j,
							"timestamp":    time.Now().Unix(),
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
		assert.Equal(t, expectedTotal, sp.stats.TotalRecords)
		assert.Equal(t, expectedTotal, sp.stats.ProcessedOK)
		assert.Equal(t, int64(0), sp.stats.ProcessedError)
	})
}

// Benchmark tests for performance validation
func BenchmarkSimpleProcessor_Execute(b *testing.B) {
	sp := &SimpleProcessor{}
	ctx := context.Background()

	err := sp.Initialize(ctx, map[string]interface{}{})
	if err != nil {
		b.Fatal(err)
	}

	testData := map[string]interface{}{
		"data": map[string]interface{}{
			"id":   12345,
			"name": "benchmark_item",
			"type": "test",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sp.Execute(ctx, testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSimpleProcessor_ProcessArray(b *testing.B) {
	sp := &SimpleProcessor{}
	testArray := make([]interface{}, 100)
	for i := range testArray {
		testArray[i] = map[string]interface{}{
			"id":   i,
			"name": "item",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sp.processArray(testArray)
	}
}

func BenchmarkSimpleProcessor_ProcessMap(b *testing.B) {
	sp := &SimpleProcessor{}
	testMap := map[string]interface{}{
		"id":     12345,
		"name":   "benchmark_item",
		"type":   "test",
		"value":  42.5,
		"active": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sp.processMap(testMap)
	}
}
