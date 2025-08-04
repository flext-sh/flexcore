package infrastructure

import (
	"fmt"

	"github.com/flext-sh/flexcore/pkg/config"
	"github.com/flext-sh/flexcore/pkg/logging"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseConnection represents a database connection
type DatabaseConnection struct {
	DB     *gorm.DB
	logger logging.LoggerInterface
}

// NewDatabaseConnection creates a new database connection
func NewDatabaseConnection(cfg *config.Config, log logging.LoggerInterface) (*DatabaseConnection, error) {
	var dialector gorm.Dialector

	switch cfg.Database.Type {
	case "postgres", "postgresql":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Name,
			cfg.Database.SSLMode)
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Use silent to avoid conflicts with our structured logging
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.MaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	log.Info("Database connection established successfully",
		zap.String("type", cfg.Database.Type),
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Name))

	return &DatabaseConnection{
		DB:     db,
		logger: log,
	}, nil
}

// Close closes the database connection
func (dc *DatabaseConnection) Close() error {
	sqlDB, err := dc.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	dc.logger.Info("Database connection closed")
	return nil
}

// Migrate runs database migrations
func (dc *DatabaseConnection) Migrate(models ...interface{}) error {
	for _, model := range models {
		if err := dc.DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate model %T: %w", model, err)
		}
	}

	dc.logger.Info("Database migrations completed successfully",
		zap.Int("models_migrated", len(models)))

	return nil
}
