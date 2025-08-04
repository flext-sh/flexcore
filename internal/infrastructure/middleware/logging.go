// Package middleware provides shared HTTP middleware functions for FlexCore servers
package middleware

import (
	"time"

	"github.com/flext-sh/flexcore/pkg/logging"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingMiddleware provides request logging middleware (DRY principle - shared across servers)
func LoggingMiddleware(logger logging.LoggerInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		logger.Info("HTTP Request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("duration", duration.String()),
			zap.String("client_ip", c.ClientIP()))
	}
}
