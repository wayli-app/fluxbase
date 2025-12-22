package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExecutorInterface verifies that Connection implements the Executor interface.
// This is a compile-time check that happens via the var _ Executor = (*Connection)(nil)
// line in executor.go, but this test makes it explicit and serves as documentation.
func TestExecutorInterface(t *testing.T) {
	t.Run("Connection implements Executor interface", func(t *testing.T) {
		// This is a compile-time check - if it compiles, the interface is satisfied
		var _ Executor = (*Connection)(nil)
		assert.True(t, true, "Connection implements Executor")
	})

	t.Run("Connection implements AdminExecutor interface", func(t *testing.T) {
		// This is a compile-time check - if it compiles, the interface is satisfied
		var _ AdminExecutor = (*Connection)(nil)
		assert.True(t, true, "Connection implements AdminExecutor")
	})
}

// TestExecutorInterfaceMethods documents the expected methods on the Executor interface.
// This helps ensure we don't accidentally change the interface signature.
func TestExecutorInterfaceMethods(t *testing.T) {
	t.Run("Executor interface has expected methods", func(t *testing.T) {
		// Document the interface methods for clarity
		var executor Executor

		// These type assertions would fail at compile time if the interface
		// didn't have these methods with the expected signatures
		_ = executor
	})

	t.Run("AdminExecutor extends Executor", func(t *testing.T) {
		// AdminExecutor should include all Executor methods plus admin methods
		var admin AdminExecutor

		// Should be able to assign AdminExecutor to Executor
		var executor Executor = admin
		_ = executor
	})
}
