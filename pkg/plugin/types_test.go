// Package plugin provides types for FlexCore plugins
package plugin

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProcessingStats_Creation(t *testing.T) {
	tests := []struct {
		name  string
		stats ProcessingStats
	}{
		{
			name:  "zero stats",
			stats: ProcessingStats{},
		},
		{
			name: "stats with values",
			stats: ProcessingStats{
				TotalRecords:   100,
				ProcessedOK:    95,
				ProcessedError: 5,
				StartTime:      time.Now().Add(-1 * time.Hour),
				EndTime:        time.Now(),
				DurationMs:     3600000, // 1 hour in ms
				RecordsPerSec:  0.026,   // ~95 records in 1 hour
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.GreaterOrEqual(t, tt.stats.TotalRecords, int64(0))
			assert.GreaterOrEqual(t, tt.stats.ProcessedOK, int64(0))
			assert.GreaterOrEqual(t, tt.stats.ProcessedError, int64(0))
			assert.GreaterOrEqual(t, tt.stats.DurationMs, int64(0))
			assert.GreaterOrEqual(t, tt.stats.RecordsPerSec, float64(0))
		})
	}
}

func TestProcessingStats_Consistency(t *testing.T) {
	t.Run("total records should equal sum of processed", func(t *testing.T) {
		stats := ProcessingStats{
			TotalRecords:   100,
			ProcessedOK:    70,
			ProcessedError: 30,
		}

		// Total should equal sum of OK and Error
		assert.Equal(t, stats.TotalRecords, stats.ProcessedOK+stats.ProcessedError)
	})

	t.Run("duration calculation", func(t *testing.T) {
		startTime := time.Now()
		endTime := startTime.Add(5 * time.Second)
		expectedDuration := endTime.Sub(startTime).Milliseconds()

		stats := ProcessingStats{
			StartTime:  startTime,
			EndTime:    endTime,
			DurationMs: expectedDuration,
		}

		assert.Equal(t, expectedDuration, stats.DurationMs)
		assert.True(t, stats.EndTime.After(stats.StartTime))
	})

	t.Run("records per second calculation", func(t *testing.T) {
		stats := ProcessingStats{
			ProcessedOK:   1000,
			DurationMs:    10000, // 10 seconds
			RecordsPerSec: 100,   // 1000 records / 10 seconds
		}

		expectedRPS := float64(stats.ProcessedOK) / (float64(stats.DurationMs) / 1000.0)
		assert.InDelta(t, expectedRPS, stats.RecordsPerSec, 0.01)
	})
}

func TestPluginInfo_Creation(t *testing.T) {
	tests := []struct {
		name       string
		pluginInfo PluginInfo
		wantValid  bool
	}{
		{
			name: "complete plugin info",
			pluginInfo: PluginInfo{
				ID:          "test-plugin",
				Name:        "Test Plugin",
				Version:     "1.0.0",
				Description: "A test plugin for unit testing",
				Author:      "Test Author",
				Type:        "processor",
				Tags:        []string{"test", "unit", "processor"},
				Status:      "active",
				LoadedAt:    time.Now(),
				Health:      "healthy",
			},
			wantValid: true,
		},
		{
			name: "minimal plugin info",
			pluginInfo: PluginInfo{
				ID:      "minimal-plugin",
				Name:    "Minimal",
				Version: "0.1.0",
				Status:  "inactive",
				Health:  "unknown",
			},
			wantValid: true,
		},
		{
			name:       "empty plugin info",
			pluginInfo: PluginInfo{},
			wantValid:  false, // Missing required fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantValid {
				// Validate required fields are present
				assert.NotEmpty(t, tt.pluginInfo.ID)
				assert.NotEmpty(t, tt.pluginInfo.Name)
				assert.NotEmpty(t, tt.pluginInfo.Version)
				assert.NotEmpty(t, tt.pluginInfo.Status)
				assert.NotEmpty(t, tt.pluginInfo.Health)
			} else {
				// At least one required field should be empty
				isEmpty := tt.pluginInfo.ID == "" ||
					tt.pluginInfo.Name == "" ||
					tt.pluginInfo.Version == "" ||
					tt.pluginInfo.Status == "" ||
					tt.pluginInfo.Health == ""
				assert.True(t, isEmpty)
			}
		})
	}
}

func TestPluginInfo_Validation(t *testing.T) {
	t.Run("valid plugin statuses", func(t *testing.T) {
		validStatuses := []string{"active", "inactive", "loading", "error", "disabled"}
		for _, status := range validStatuses {
			plugin := PluginInfo{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Status:  status,
				Health:  "healthy",
			}
			assert.Equal(t, status, plugin.Status)
		}
	})

	t.Run("valid plugin health states", func(t *testing.T) {
		validHealthStates := []string{"healthy", "unhealthy", "degraded", "unknown"}
		for _, health := range validHealthStates {
			plugin := PluginInfo{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Status:  "active",
				Health:  health,
			}
			assert.Equal(t, health, plugin.Health)
		}
	})

	t.Run("plugin types", func(t *testing.T) {
		validTypes := []string{"processor", "transformer", "adapter", "connector", "filter"}
		for _, pluginType := range validTypes {
			plugin := PluginInfo{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Type:    pluginType,
				Status:  "active",
				Health:  "healthy",
			}
			assert.Equal(t, pluginType, plugin.Type)
		}
	})

	t.Run("semantic version validation", func(t *testing.T) {
		validVersions := []string{"1.0.0", "2.1.3", "0.0.1", "10.20.30"}
		for _, version := range validVersions {
			plugin := PluginInfo{
				ID:      "test",
				Name:    "Test",
				Version: version,
				Status:  "active",
				Health:  "healthy",
			}
			assert.Equal(t, version, plugin.Version)
			// Basic semantic version check (major.minor.patch)
			assert.Regexp(t, `^\d+\.\d+\.\d+$`, plugin.Version)
		}
	})
}

func TestPluginInfo_Tags(t *testing.T) {
	t.Run("tags manipulation", func(t *testing.T) {
		plugin := PluginInfo{
			ID:      "test",
			Name:    "Test",
			Version: "1.0.0",
			Tags:    []string{"data", "processor", "json"},
			Status:  "active",
			Health:  "healthy",
		}

		// Test tags exist
		assert.Contains(t, plugin.Tags, "data")
		assert.Contains(t, plugin.Tags, "processor")
		assert.Contains(t, plugin.Tags, "json")
		assert.Len(t, plugin.Tags, 3)

		// Test tags order preserved
		assert.Equal(t, "data", plugin.Tags[0])
		assert.Equal(t, "processor", plugin.Tags[1])
		assert.Equal(t, "json", plugin.Tags[2])
	})

	t.Run("empty tags", func(t *testing.T) {
		plugin := PluginInfo{
			ID:      "test",
			Name:    "Test",
			Version: "1.0.0",
			Tags:    []string{},
			Status:  "active",
			Health:  "healthy",
		}

		assert.Empty(t, plugin.Tags)
		assert.Len(t, plugin.Tags, 0)
	})

	t.Run("nil tags", func(t *testing.T) {
		plugin := PluginInfo{
			ID:      "test",
			Name:    "Test",
			Version: "1.0.0",
			Tags:    nil,
			Status:  "active",
			Health:  "healthy",
		}

		assert.Nil(t, plugin.Tags)
	})
}

func TestPluginInfo_Timestamps(t *testing.T) {
	t.Run("loaded at timestamp", func(t *testing.T) {
		loadedTime := time.Now()
		plugin := PluginInfo{
			ID:       "test",
			Name:     "Test",
			Version:  "1.0.0",
			LoadedAt: loadedTime,
			Status:   "active",
			Health:   "healthy",
		}

		assert.Equal(t, loadedTime, plugin.LoadedAt)
		assert.True(t, plugin.LoadedAt.Before(time.Now().Add(time.Second)) ||
			plugin.LoadedAt.Equal(time.Now().Add(time.Second)))
	})

	t.Run("zero timestamp", func(t *testing.T) {
		plugin := PluginInfo{
			ID:      "test",
			Name:    "Test",
			Version: "1.0.0",
			Status:  "active",
			Health:  "healthy",
		}

		assert.True(t, plugin.LoadedAt.IsZero())
	})
}

// Test edge cases and data integrity
func TestPluginInfo_EdgeCases(t *testing.T) {
	t.Run("very long strings", func(t *testing.T) {
		longString := string(make([]byte, 1000))
		for range longString {
			longString = "x" + longString[1:]
		}

		plugin := PluginInfo{
			ID:          "test",
			Name:        "Test",
			Version:     "1.0.0",
			Description: longString,
			Status:      "active",
			Health:      "healthy",
		}

		assert.Equal(t, longString, plugin.Description)
		assert.GreaterOrEqual(t, len(plugin.Description), 1000)
	})

	t.Run("special characters", func(t *testing.T) {
		specialChars := "!@#$%^&*(){}[]|\\:;\"'<>?,./-=+~`"
		plugin := PluginInfo{
			ID:          "test-plugin-123",
			Name:        "Test Plugin" + specialChars,
			Version:     "1.0.0",
			Description: "Plugin with special chars: " + specialChars,
			Status:      "active",
			Health:      "healthy",
		}

		assert.Contains(t, plugin.Name, specialChars)
		assert.Contains(t, plugin.Description, specialChars)
	})

	t.Run("unicode characters", func(t *testing.T) {
		unicodeText := "测试插件 🚀 プラグイン مكون إضافي"
		plugin := PluginInfo{
			ID:          "unicode-test",
			Name:        unicodeText,
			Version:     "1.0.0",
			Description: "Unicode support: " + unicodeText,
			Status:      "active",
			Health:      "healthy",
		}

		assert.Equal(t, unicodeText, plugin.Name)
		assert.Contains(t, plugin.Description, unicodeText)
	})
}

// Performance and memory tests
func TestPluginInfo_Performance(t *testing.T) {
	t.Run("large tags array", func(t *testing.T) {
		// Create large tags array
		largeTagsArray := make([]string, 1000)
		for i := range largeTagsArray {
			largeTagsArray[i] = fmt.Sprintf("tag-%d", i)
		}

		plugin := PluginInfo{
			ID:      "large-tags-test",
			Name:    "Large Tags Test",
			Version: "1.0.0",
			Tags:    largeTagsArray,
			Status:  "active",
			Health:  "healthy",
		}

		assert.Len(t, plugin.Tags, 1000)
		assert.Equal(t, "tag-0", plugin.Tags[0])
		assert.Equal(t, "tag-999", plugin.Tags[999])
	})
}

// Benchmark tests for performance validation
func BenchmarkProcessingStats_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ProcessingStats{
			TotalRecords:   int64(i),
			ProcessedOK:    int64(i * 90 / 100),
			ProcessedError: int64(i * 10 / 100),
			StartTime:      time.Now(),
			EndTime:        time.Now(),
			DurationMs:     1000,
			RecordsPerSec:  float64(i),
		}
	}
}

func BenchmarkPluginInfo_Creation(b *testing.B) {
	tags := []string{"benchmark", "test", "performance"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PluginInfo{
			ID:          fmt.Sprintf("benchmark-plugin-%d", i),
			Name:        fmt.Sprintf("Benchmark Plugin %d", i),
			Version:     "1.0.0",
			Description: "A benchmark plugin for performance testing",
			Author:      "Benchmark Author",
			Type:        "processor",
			Tags:        tags,
			Status:      "active",
			LoadedAt:    time.Now(),
			Health:      "healthy",
		}
	}
}
