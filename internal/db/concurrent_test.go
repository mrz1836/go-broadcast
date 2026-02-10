package db

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestConcurrentReads(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, db.Create(group).Error)

	// Create multiple targets
	for i := 0; i < 10; i++ {
		target := &Target{
			GroupID:  group.ID,
			Repo:     fmt.Sprintf("mrz1836/repo%d", i),
			Position: i,
		}
		require.NoError(t, db.Create(target).Error)
	}

	repo := NewGroupRepository(db)

	// Run 100 concurrent reads
	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.GetByID(ctx, group.ID)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent read failed: %v", err)
	}
}

func TestConcurrentWrites_DifferentGroups(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create test config
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	repo := NewGroupRepository(db)

	// Create 20 groups concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 20)
	groupIDs := make([]uint, 20)
	var mu sync.Mutex

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			group := &Group{
				ConfigID:   config.ID,
				ExternalID: fmt.Sprintf("group-%d", idx),
				Name:       fmt.Sprintf("Group %d", idx),
				Position:   idx,
			}
			err := repo.Create(ctx, group)
			if err != nil {
				errChan <- err
				return
			}
			mu.Lock()
			groupIDs[idx] = group.ID
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent write failed: %v", err)
	}

	// Verify all groups were created
	groups, err := repo.List(ctx, config.ID)
	require.NoError(t, err)
	assert.Len(t, groups, 20)
}

func TestConcurrentWrites_DifferentTargets(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, db.Create(group).Error)

	repo := NewTargetRepository(db)

	// Create 50 targets concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			target := &Target{
				GroupID:  group.ID,
				Repo:     fmt.Sprintf("mrz1836/repo-%d", idx),
				Position: idx,
			}
			err := repo.Create(ctx, target)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent target write failed: %v", err)
	}

	// Verify all targets were created
	targets, err := repo.List(ctx, group.ID)
	require.NoError(t, err)
	assert.Len(t, targets, 50)
}

func TestConcurrentReadWrite(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, db.Create(group).Error)

	// Create initial targets
	for i := 0; i < 5; i++ {
		target := &Target{
			GroupID:  group.ID,
			Repo:     fmt.Sprintf("mrz1836/initial-%d", i),
			Position: i,
		}
		require.NoError(t, db.Create(target).Error)
	}

	repo := NewTargetRepository(db)

	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	// 50 concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.List(ctx, group.ID)
			if err != nil {
				errChan <- err
			}
		}()
	}

	// 10 concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			target := &Target{
				GroupID:  group.ID,
				Repo:     fmt.Sprintf("mrz1836/concurrent-%d", idx),
				Position: 100 + idx,
			}
			err := repo.Create(ctx, target)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent read/write failed: %v", err)
	}

	// Verify final state
	targets, err := repo.List(ctx, group.ID)
	require.NoError(t, err)
	assert.Len(t, targets, 15) // 5 initial + 10 concurrent
}

func TestConcurrentUpdates(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
		Priority:   0,
	}
	require.NoError(t, db.Create(group).Error)

	repo := NewGroupRepository(db)

	// 20 concurrent updates to the same group
	var wg sync.WaitGroup
	errChan := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			
			// Read current state
			g, err := repo.GetByID(ctx, group.ID)
			if err != nil {
				errChan <- err
				return
			}

			// Update priority
			g.Priority = idx
			err = repo.Update(ctx, g)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent update failed: %v", err)
	}

	// Verify final state (should have one of the priorities)
	finalGroup, err := repo.GetByID(ctx, group.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, finalGroup.Priority, 0)
	assert.Less(t, finalGroup.Priority, 20)
}

func TestConcurrentRefManagement(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, db.Create(group).Error)

	target := &Target{
		GroupID: group.ID,
		Repo:    "mrz1836/test-repo",
	}
	require.NoError(t, db.Create(target).Error)

	// Create multiple file lists
	fileLists := make([]*FileList, 10)
	for i := 0; i < 10; i++ {
		fileLists[i] = &FileList{
			ConfigID:   config.ID,
			ExternalID: fmt.Sprintf("list-%d", i),
			Name:       fmt.Sprintf("List %d", i),
		}
		require.NoError(t, db.Create(fileLists[i]).Error)
	}

	repo := NewTargetRepository(db)

	// Add refs concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := repo.AddFileListRef(ctx, target.ID, fileLists[idx].ID, idx)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent ref add failed: %v", err)
	}

	// Verify all refs were created
	var refCount int64
	err := db.Model(&TargetFileListRef{}).
		Where("target_id = ?", target.ID).
		Count(&refCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(10), refCount)
}

func TestConcurrentQuery(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Create test data
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{
		ConfigID:   config.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, db.Create(group).Error)

	// Create targets with file mappings
	for i := 0; i < 5; i++ {
		target := &Target{
			GroupID:  group.ID,
			Repo:     fmt.Sprintf("mrz1836/repo-%d", i),
			Position: i,
		}
		require.NoError(t, db.Create(target).Error)

		// Create file mapping
		fm := &FileMapping{
			OwnerType: "target",
			OwnerID:   target.ID,
			Src:       "README.md",
			Dest:      "README.md",
			Position:  0,
		}
		require.NoError(t, db.Create(fm).Error)
	}

	repo := NewQueryRepository(db)

	// Run 50 concurrent queries
	var wg sync.WaitGroup
	errChan := make(chan error, 50)
	resultCounts := make([]int, 50)
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			targets, err := repo.FindByFile(ctx, "README.md")
			if err != nil {
				errChan <- err
				return
			}
			mu.Lock()
			resultCounts[idx] = len(targets)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent query failed: %v", err)
	}

	// Verify all queries returned consistent results
	for i, count := range resultCounts {
		assert.Equal(t, 5, count, "Query %d returned inconsistent results", i)
	}
}

func TestConcurrentTransactions(t *testing.T) {
	db := TestDB(t)

	// Create test config
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	// Run 10 concurrent transactions, each creating a group with associated data
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Each transaction creates a group with source, targets, and file mappings
			err := db.Transaction(func(tx *gorm.DB) error {
				group := &Group{
					ConfigID:   config.ID,
					ExternalID: fmt.Sprintf("group-%d", idx),
					Name:       fmt.Sprintf("Group %d", idx),
					Source: Source{
						Repo:   fmt.Sprintf("mrz1836/source-%d", idx),
						Branch: "main",
					},
				}
				if err := tx.Create(group).Error; err != nil {
					return err
				}

				// Create target
				target := &Target{
					GroupID: group.ID,
					Repo:    fmt.Sprintf("mrz1836/target-%d", idx),
				}
				if err := tx.Create(target).Error; err != nil {
					return err
				}

				// Create file mapping
				fm := &FileMapping{
					OwnerType: "target",
					OwnerID:   target.ID,
					Src:       "README.md",
					Dest:      "README.md",
				}
				if err := tx.Create(fm).Error; err != nil {
					return err
				}

				return nil
			})

			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent transaction failed: %v", err)
	}

	// Verify all data was created
	var groups []*Group
	err := db.Preload("Source").Preload("Targets").Where("config_id = ?", config.ID).Find(&groups).Error
	require.NoError(t, err)
	assert.Len(t, groups, 10)

	for _, g := range groups {
		assert.NotEmpty(t, g.Source.Repo)
		assert.Len(t, g.Targets, 1)
	}
}
