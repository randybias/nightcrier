package storage

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/file"
)

// MigrationConfig holds configuration for database migrations
type MigrationConfig struct {
	// MigrationsPath is the path to the migrations directory
	MigrationsPath string
	// DatabaseType is either "sqlite" or "postgres"
	DatabaseType string
	// DatabasePath is the path to the SQLite database file (sqlite only)
	DatabasePath string
	// DatabaseURL is the PostgreSQL connection string (postgres only)
	DatabaseURL string
}

// RunMigrations executes all pending database migrations
func RunMigrations(cfg *MigrationConfig) error {
	// Open database connection
	db, err := openDatabase(cfg)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create migration driver based on database type
	driver, err := createMigrationDriver(db, cfg.DatabaseType)
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create file source for migrations
	migrationsPath := cfg.MigrationsPath
	if !filepath.IsAbs(migrationsPath) {
		// Convert to absolute path if relative
		absPath, err := filepath.Abs(migrationsPath)
		if err != nil {
			return fmt.Errorf("failed to resolve migrations path: %w", err)
		}
		migrationsPath = absPath
	}

	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	sourceInstance, err := (&file.File{}).Open(sourceURL)
	if err != nil {
		return fmt.Errorf("failed to open migrations source: %w", err)
	}

	// Create migration instance
	m, err := migrate.NewWithInstance(
		"file",
		sourceInstance,
		cfg.DatabaseType,
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// RollbackMigrations rolls back database migrations by the specified number of steps
// If steps is 0, all migrations are rolled back
func RollbackMigrations(cfg *MigrationConfig, steps int) error {
	// Open database connection
	db, err := openDatabase(cfg)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create migration driver based on database type
	driver, err := createMigrationDriver(db, cfg.DatabaseType)
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create file source for migrations
	migrationsPath := cfg.MigrationsPath
	if !filepath.IsAbs(migrationsPath) {
		// Convert to absolute path if relative
		absPath, err := filepath.Abs(migrationsPath)
		if err != nil {
			return fmt.Errorf("failed to resolve migrations path: %w", err)
		}
		migrationsPath = absPath
	}

	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	sourceInstance, err := (&file.File{}).Open(sourceURL)
	if err != nil {
		return fmt.Errorf("failed to open migrations source: %w", err)
	}

	// Create migration instance
	m, err := migrate.NewWithInstance(
		"file",
		sourceInstance,
		cfg.DatabaseType,
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Rollback migrations
	if steps == 0 {
		// Rollback all migrations
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to rollback all migrations: %w", err)
		}
	} else {
		// Rollback specific number of steps
		if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to rollback %d migration(s): %w", steps, err)
		}
	}

	return nil
}

// GetMigrationVersion returns the current migration version
func GetMigrationVersion(cfg *MigrationConfig) (uint, bool, error) {
	// Open database connection
	db, err := openDatabase(cfg)
	if err != nil {
		return 0, false, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create migration driver based on database type
	driver, err := createMigrationDriver(db, cfg.DatabaseType)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create file source for migrations
	migrationsPath := cfg.MigrationsPath
	if !filepath.IsAbs(migrationsPath) {
		// Convert to absolute path if relative
		absPath, err := filepath.Abs(migrationsPath)
		if err != nil {
			return 0, false, fmt.Errorf("failed to resolve migrations path: %w", err)
		}
		migrationsPath = absPath
	}

	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	sourceInstance, err := (&file.File{}).Open(sourceURL)
	if err != nil {
		return 0, false, fmt.Errorf("failed to open migrations source: %w", err)
	}

	// Create migration instance
	m, err := migrate.NewWithInstance(
		"file",
		sourceInstance,
		cfg.DatabaseType,
		driver,
	)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Get current version
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// openDatabase opens a database connection based on the configuration
func openDatabase(cfg *MigrationConfig) (*sql.DB, error) {
	switch cfg.DatabaseType {
	case "sqlite":
		if cfg.DatabasePath == "" {
			return nil, fmt.Errorf("database path is required for SQLite")
		}
		db, err := sql.Open("sqlite", cfg.DatabasePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite database: %w", err)
		}
		return db, nil

	case "postgres":
		if cfg.DatabaseURL == "" {
			return nil, fmt.Errorf("database URL is required for PostgreSQL")
		}
		db, err := sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
		}
		return db, nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DatabaseType)
	}
}

// createMigrationDriver creates a migration driver for the specified database type
func createMigrationDriver(db *sql.DB, dbType string) (database.Driver, error) {
	switch dbType {
	case "sqlite":
		driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite migration driver: %w", err)
		}
		return driver, nil

	case "postgres":
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to create PostgreSQL migration driver: %w", err)
		}
		return driver, nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
