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

// RunMigrations registers and applies all data migrations.
// Called after AutoMigrate to handle data transformations that GORM's
// AutoMigrate cannot express (e.g., data copy, FK re-pointing, table drops).
func RunMigrations(db *gorm.DB) error {
	mgr := NewMigrationManager(db)

	// Register all migrations
	mgr.Register(migrationConsolidateAnalyticsRepos())

	return mgr.Apply()
}

// migrationConsolidateAnalyticsRepos merges analytics_repositories into repos.
// Steps:
// 1. Populate full_name on existing repos
// 2. Copy analytics-only fields from analytics_repositories into matching repos
// 3. Re-point repository_snapshots, security_alerts, ci_metrics_snapshots FKs
// 4. Drop analytics_repositories table
func migrationConsolidateAnalyticsRepos() Migration {
	return Migration{
		Version:     "20260220_001",
		Description: "Consolidate analytics_repositories into repos table",
		Up: func(tx *gorm.DB) error {
			// Check if analytics_repositories table exists; if not, nothing to migrate
			if !tx.Migrator().HasTable("analytics_repositories") {
				// Still populate full_name for existing repos
				if err := tx.Exec(`
					UPDATE repos
					SET full_name = (
						SELECT o.name || '/' || repos.name
						FROM organizations o
						WHERE o.id = repos.organization_id
					)
					WHERE full_name IS NULL OR full_name = ''
				`).Error; err != nil {
					return fmt.Errorf("failed to populate full_name: %w", err)
				}
				return nil
			}

			// 1. Populate full_name on existing repos from organization name
			if err := tx.Exec(`
				UPDATE repos
				SET full_name = (
					SELECT o.name || '/' || repos.name
					FROM organizations o
					WHERE o.id = repos.organization_id
				)
				WHERE full_name IS NULL OR full_name = ''
			`).Error; err != nil {
				return fmt.Errorf("failed to populate full_name: %w", err)
			}

			// 2. Copy analytics fields into matching repos rows
			// GORM's default naming strategy splits "ETag" into "e_tag",
			// so the physical column names are metadata_e_tag / security_e_tag.
			if err := tx.Exec(`
				UPDATE repos
				SET
					metadata_e_tag = COALESCE((
						SELECT ar.metadata_e_tag FROM analytics_repositories ar
						WHERE ar.full_name = repos.full_name
					), repos.metadata_e_tag),
					security_e_tag = COALESCE((
						SELECT ar.security_e_tag FROM analytics_repositories ar
						WHERE ar.full_name = repos.full_name
					), repos.security_e_tag),
					last_sync_at = COALESCE((
						SELECT ar.last_sync_at FROM analytics_repositories ar
						WHERE ar.full_name = repos.full_name
					), repos.last_sync_at),
					last_sync_run_id = COALESCE((
						SELECT ar.last_sync_run_id FROM analytics_repositories ar
						WHERE ar.full_name = repos.full_name
					), repos.last_sync_run_id)
				WHERE EXISTS (
					SELECT 1 FROM analytics_repositories ar
					WHERE ar.full_name = repos.full_name
				)
			`).Error; err != nil {
				return fmt.Errorf("failed to copy analytics fields: %w", err)
			}

			// 3. Re-point repository_snapshots FK from analytics_repositories IDs to repos IDs
			if err := tx.Exec(`
				UPDATE repository_snapshots
				SET repository_id = (
					SELECT r.id FROM repos r
					JOIN analytics_repositories ar ON ar.full_name = r.full_name
					WHERE ar.id = repository_snapshots.repository_id
				)
				WHERE EXISTS (
					SELECT 1 FROM analytics_repositories ar
					JOIN repos r ON ar.full_name = r.full_name
					WHERE ar.id = repository_snapshots.repository_id
				)
			`).Error; err != nil {
				return fmt.Errorf("failed to re-point repository_snapshots: %w", err)
			}

			// 4. Re-point security_alerts FK
			if err := tx.Exec(`
				UPDATE security_alerts
				SET repository_id = (
					SELECT r.id FROM repos r
					JOIN analytics_repositories ar ON ar.full_name = r.full_name
					WHERE ar.id = security_alerts.repository_id
				)
				WHERE EXISTS (
					SELECT 1 FROM analytics_repositories ar
					JOIN repos r ON ar.full_name = r.full_name
					WHERE ar.id = security_alerts.repository_id
				)
			`).Error; err != nil {
				return fmt.Errorf("failed to re-point security_alerts: %w", err)
			}

			// 5. Re-point ci_metrics_snapshots FK
			if err := tx.Exec(`
				UPDATE ci_metrics_snapshots
				SET repository_id = (
					SELECT r.id FROM repos r
					JOIN analytics_repositories ar ON ar.full_name = r.full_name
					WHERE ar.id = ci_metrics_snapshots.repository_id
				)
				WHERE EXISTS (
					SELECT 1 FROM analytics_repositories ar
					JOIN repos r ON ar.full_name = r.full_name
					WHERE ar.id = ci_metrics_snapshots.repository_id
				)
			`).Error; err != nil {
				return fmt.Errorf("failed to re-point ci_metrics_snapshots: %w", err)
			}

			// 6. Drop analytics_repositories table
			if err := tx.Exec("DROP TABLE IF EXISTS analytics_repositories").Error; err != nil {
				return fmt.Errorf("failed to drop analytics_repositories: %w", err)
			}

			return nil
		},
		Down: nil, // One-way migration; rollback not supported
	}
}
