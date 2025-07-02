// Package core - DEPRECATED: Use internal/domain instead
//
// This package is deprecated and will be removed in v2.0.0.
// Please migrate to github.com/flext/flexcore/internal/domain
package core

import (
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/flext/flexcore/internal/domain"
)

var (
	deprecationWarnings sync.Map
	deprecationOnce     sync.Once
)

// DEPRECATED: Use internal/domain.FlexCore instead
// This type will be removed in v2.0.0
type FlexCore = domain.FlexCore

// DEPRECATED: Use internal/domain.FlexCoreConfig instead
// This type will be removed in v2.0.0
type FlexCoreConfig = domain.FlexCoreConfig

// DEPRECATED: Use internal/domain.Event instead
// This type will be removed in v2.0.0
type Event = domain.Event

// DEPRECATED: Use internal/domain.Message instead
// This type will be removed in v2.0.0
type Message = domain.Message

// Note: Generic type aliases require Go 1.23+
// For now, we'll use a struct wrapper for compatibility

func init() {
	// Show deprecation warning once per process
	deprecationOnce.Do(func() {
		if os.Getenv("FLEXCORE_HIDE_DEPRECATION_WARNINGS") != "true" {
			log.Printf("🚨 DEPRECATION WARNING: Package 'core' is deprecated. Use 'internal/domain' instead.")
			log.Printf("   This package will be removed in FlexCore v2.0.0")
			log.Printf("   Set FLEXCORE_HIDE_DEPRECATION_WARNINGS=true to hide this warning")
			log.Printf("   Migration guide: https://github.com/flext/flexcore/docs/migration-v2.md")
		}
	})
}

// logDeprecatedUsage logs when a deprecated function is used
func logDeprecatedUsage(functionName string) {
	// Only warn once per function per process
	if _, warned := deprecationWarnings.LoadOrStore(functionName, true); !warned {
		_, file, line, _ := runtime.Caller(2)
		log.Printf("🚨 DEPRECATED: %s used at %s:%d - migrate to internal/domain", functionName, file, line)
	}
}

// DEPRECATED: Use internal/domain.NewFlexCore instead
// This function will be removed in v2.0.0
func NewFlexCore(config *FlexCoreConfig) *domain.Result[*FlexCore] {
	logDeprecatedUsage("NewFlexCore")
	return domain.NewFlexCore(config)
}

// DEPRECATED: Use internal/domain.DefaultConfig instead
// This function will be removed in v2.0.0
func DefaultConfig() *FlexCoreConfig {
	logDeprecatedUsage("DefaultConfig")
	return domain.DefaultConfig()
}

// DEPRECATED: Use internal/domain.NewEvent instead
// This function will be removed in v2.0.0
func NewEvent(eventType, aggregateID string, data map[string]interface{}) *Event {
	logDeprecatedUsage("NewEvent")
	return domain.NewEvent(eventType, aggregateID, data)
}

// DEPRECATED: Use internal/domain.NewMessage instead
// This function will be removed in v2.0.0
func NewMessage(queue string, data map[string]interface{}) *Message {
	logDeprecatedUsage("NewMessage")
	return domain.NewMessage(queue, data)
}

// DEPRECATED: Use internal/domain.NewSuccess instead
// This function will be removed in v2.0.0
func NewSuccess[T any](value T) *domain.Result[T] {
	logDeprecatedUsage("NewSuccess")
	return domain.NewSuccess(value)
}

// DEPRECATED: Use internal/domain.NewFailure instead
// This function will be removed in v2.0.0
func NewFailure[T any](err error) *domain.Result[T] {
	logDeprecatedUsage("NewFailure")
	return domain.NewFailure[T](err)
}