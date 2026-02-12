package db

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Migration represents a single database migration
type Migration struct {
	Version     string
	Description string
	Up          func(*gorm.DB) error
	Down        func(*gorm.DB) error
}

// MigrationManager handles schema migrations
type MigrationManager struct {
	db             *gorm.DB
	mu             sync.Mutex // Protects migrations slice
	opMu           sync.Mutex // Serializes Apply/Rollback operations
	migrations     []Migration
	schemaInitOnce sync.Once
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *gorm.DB) *MigrationManager {
	return &MigrationManager{
		db:         db,
		migrations: []Migration{},
	}
}

// Register adds a migration to the manager
func (m *MigrationManager) Register(migration Migration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.migrations = append(m.migrations, migration)
}

// AppliedMigrations returns all applied migration versions
func (m *MigrationManager) AppliedMigrations() ([]string, error) {
	var migrations []SchemaMigration
	if err := m.db.Order("applied_at ASC").Find(&migrations).Error; err != nil {
		return nil, err
	}

	versions := make([]string, len(migrations))
	for i, migration := range migrations {
		versions[i] = migration.Version
	}
	return versions, nil
}

// IsMigrationApplied checks if a specific migration version has been applied
func (m *MigrationManager) IsMigrationApplied(version string) (bool, error) {
	var count int64
	err := m.db.Model(&SchemaMigration{}).Where("version = ?", version).Count(&count).Error
	return count > 0, err
}

// Apply runs all pending migrations
func (m *MigrationManager) Apply() error {
	// Serialize all Apply() operations
	m.opMu.Lock()
	defer m.opMu.Unlock()

	// Ensure schema_migrations table exists (thread-safe, only runs once)
	var schemaErr error
	m.schemaInitOnce.Do(func() {
		schemaErr = m.db.AutoMigrate(&SchemaMigration{})
	})
	if schemaErr != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", schemaErr)
	}

	// Get a snapshot of migrations under lock
	m.mu.Lock()
	migrations := make([]Migration, len(m.migrations))
	copy(migrations, m.migrations)
	m.mu.Unlock()

	for _, migration := range migrations {
		// Check if migration is already applied
		// This is now safe because opMu prevents concurrent Apply() calls
		applied, err := m.IsMigrationApplied(migration.Version)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if applied {
			continue
		}

		// Run migration in transaction
		if err := m.db.Transaction(func(tx *gorm.DB) error {
			// Double-check inside transaction for defense-in-depth
			// Handles edge cases where another process (not goroutine) might have applied
			var count int64
			if err := tx.Model(&SchemaMigration{}).Where("version = ?", migration.Version).Count(&count).Error; err != nil {
				return fmt.Errorf("failed to verify migration status: %w", err)
			}

			if count > 0 {
				// Already applied by another process, skip
				return nil
			}

			// Execute migration
			if err := migration.Up(tx); err != nil {
				return fmt.Errorf("migration %s failed: %w", migration.Version, err)
			}

			// Calculate checksum for integrity verification
			checksum := calculateChecksum(migration.Version, migration.Description)

			// Record migration
			record := SchemaMigration{
				Version:     migration.Version,
				AppliedAt:   time.Now().UTC(),
				Description: migration.Description,
				Checksum:    checksum,
			}

			if err := tx.Create(&record).Error; err != nil {
				return fmt.Errorf("failed to record migration: %w", err)
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// Rollback reverts the last applied migration
func (m *MigrationManager) Rollback() error {
	// Serialize all Rollback() operations
	m.opMu.Lock()
	defer m.opMu.Unlock()

	// Get last applied migration
	var lastMigration SchemaMigration
	if err := m.db.Order("applied_at DESC").First(&lastMigration).Error; err != nil {
		return fmt.Errorf("no migrations to rollback: %w", err)
	}

	// Find the migration definition under lock, copy it
	m.mu.Lock()
	var migration Migration
	var found bool
	for i := range m.migrations {
		if m.migrations[i].Version == lastMigration.Version {
			migration = m.migrations[i]
			found = true
			break
		}
	}
	m.mu.Unlock()

	if !found {
		return fmt.Errorf("%w for version: %s", ErrMigrationNotFound, lastMigration.Version)
	}

	if migration.Down == nil {
		return fmt.Errorf("%w: %s", ErrNoDownFunction, migration.Version)
	}

	// Run rollback in transaction
	return m.db.Transaction(func(tx *gorm.DB) error {
		// Re-fetch and verify inside transaction
		// Ensures migration still exists and hasn't been rolled back by another goroutine
		var currentMigration SchemaMigration
		if err := tx.Order("applied_at DESC").First(&currentMigration).Error; err != nil {
			return fmt.Errorf("no migrations to rollback: %w", err)
		}

		// Verify we're still rolling back the expected migration
		if currentMigration.Version != lastMigration.Version {
			return fmt.Errorf("%w: expected %s, found %s",
				ErrMigrationStateChanged, lastMigration.Version, currentMigration.Version)
		}

		// Execute rollback
		if err := migration.Down(tx); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		// Remove migration record
		if err := tx.Delete(&currentMigration).Error; err != nil {
			return fmt.Errorf("failed to remove migration record: %w", err)
		}

		return nil
	})
}

// calculateChecksum generates a checksum for a migration
func calculateChecksum(version, description string) string {
	data := fmt.Sprintf("%s:%s", version, description)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}
