package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirectoryProcessor_WorkerCount(t *testing.T) {
	t.Parallel()

	t.Run("set and get worker count", func(t *testing.T) {
		t.Parallel()

		dp := &DirectoryProcessor{workerCount: 5}
		assert.Equal(t, 5, dp.GetWorkerCount())

		dp.SetWorkerCount(10)
		assert.Equal(t, 10, dp.GetWorkerCount())
	})

	t.Run("zero count is ignored", func(t *testing.T) {
		t.Parallel()

		dp := &DirectoryProcessor{workerCount: 5}
		dp.SetWorkerCount(0)
		assert.Equal(t, 5, dp.GetWorkerCount())
	})

	t.Run("negative count is ignored", func(t *testing.T) {
		t.Parallel()

		dp := &DirectoryProcessor{workerCount: 5}
		dp.SetWorkerCount(-1)
		assert.Equal(t, 5, dp.GetWorkerCount())
	})

	t.Run("set to one", func(t *testing.T) {
		t.Parallel()

		dp := &DirectoryProcessor{workerCount: 5}
		dp.SetWorkerCount(1)
		assert.Equal(t, 1, dp.GetWorkerCount())
	})
}
